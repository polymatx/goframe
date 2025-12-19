package rabbit

import (
	"container/ring"
	"context"
	"fmt"
	"sync"

	"github.com/polymatx/goframe/pkg/healthz"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

type Channel interface {
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Close() error
}

var (
	connRng            = make(map[string]*ring.Ring, 0)
	connRngLock        = &sync.RWMutex{}
	once               = sync.Once{}
	rng                = make(map[string]*ring.Ring, 0)
	rngLock            = &sync.RWMutex{}
	kill               context.Context
	killCancel         context.CancelFunc
	rabbitConnExpected = make([]rabbitExpected, 0)
)

var notifyClose = make(chan *amqp.Error, 10)

type ignite struct {
}

func (in *ignite) Health(ctx context.Context) error {
	select {
	case err := <-notifyClose:
		if err != nil {
			return fmt.Errorf("RabbitMQ error happen : %s", err)
		}
	default: // Do not block
	}
	return nil
}

type chnlLock struct {
	chn    Channel
	lock   *sync.Mutex
	rtrn   chan amqp.Confirmation
	wg     *sync.WaitGroup
	closed bool
}

func Initialize(ctx context.Context) {
	once.Do(func() {
		for i := range rabbitConnExpected {
			if err := initializeConnection(ctx, rabbitConnExpected[i]); err != nil {
				logrus.Errorf("failed to initialize rabbit connection: %s", err.Error())
				return
			}
		}
		healthz.Register(&ignite{})
		logrus.Info("Rabbit initialized")
	})
}

func initializeConnection(ctx context.Context, expected rabbitExpected) error {
	kill, killCancel = context.WithCancel(ctx)
	cnt := viper.GetInt("rabbit_connection_num")
	if cnt < 1 {
		cnt = 1
	}
	connString := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		expected.user,
		expected.password,
		expected.host,
		expected.port,
		expected.vHost,
	)

	connRngLock.Lock()
	rngLock.Lock()
	defer func() {
		rngLock.Unlock()
		connRngLock.Unlock()
	}()

	connRng[expected.containerName] = ring.New(cnt)
	for j := 0; j < cnt; j++ {
		c, err := amqp.Dial(connString)
		if err != nil {
			return fmt.Errorf("error connecting to rabbit: %w", err)
		}
		connRng[expected.containerName].Value = c
		connRng[expected.containerName] = connRng[expected.containerName].Next()
	}
	connRng[expected.containerName] = connRng[expected.containerName].Next()

	conn := connRng[expected.containerName].Value.(*amqp.Connection)

	chn, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("error creating channel: %w", err)
	}

	err = chn.ExchangeDeclare(
		viper.GetString("exchange_name"),
		viper.GetString("exchange_type"),
		true,
		false,
		false,
		false,
		amqp.Table{},
	)
	if err != nil {
		chn.Close()
		return fmt.Errorf("error declaring exchange: %w", err)
	}
	chn.Close()

	publishNum := viper.GetInt("rabbit_publish_num")
	if publishNum < 1 {
		publishNum = 1
	}
	rng[expected.containerName] = ring.New(publishNum)

	confirmLen := viper.GetInt("rabbit_confirm_len")
	if confirmLen < 1 {
		confirmLen = 100
	}

	for j := 0; j < publishNum; j++ {
		connRng[expected.containerName] = connRng[expected.containerName].Next()
		conn := connRng[expected.containerName].Value.(*amqp.Connection)
		pchn, err := conn.Channel()
		if err != nil {
			return fmt.Errorf("error creating publish channel: %w", err)
		}
		rtrn := make(chan amqp.Confirmation, confirmLen)
		if err = pchn.Confirm(false); err != nil {
			pchn.Close()
			return fmt.Errorf("error enabling confirm mode: %w", err)
		}
		pchn.NotifyPublish(rtrn)
		tmp := chnlLock{
			chn:    pchn,
			lock:   &sync.Mutex{},
			wg:     &sync.WaitGroup{},
			rtrn:   rtrn,
			closed: false,
		}
		go publishConfirm(&tmp)
		rng[expected.containerName].Value = &tmp
		rng[expected.containerName] = rng[expected.containerName].Next()
	}

	return nil
}

func publishConfirm(cl *chnlLock) {
	for range cl.rtrn {
		cl.wg.Done()
	}
}

// Close closes all RabbitMQ connections
func Close() {
	if killCancel != nil {
		killCancel()
	}
}

type rabbitExpected struct {
	containerName string
	host          string
	port          int
	user          string
	password      string
	vHost         string
}

func RegisterRabbit(cnt, host, user, password, vHost string, port int) {
	rabbitConnExpected = append(rabbitConnExpected, rabbitExpected{
		containerName: cnt,
		host:          host,
		vHost:         vHost,
		user:          user,
		password:      password,
		port:          port,
	})
}
