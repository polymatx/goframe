package main

import (
	"fmt"
	"net/http"

	"github.com/polymatx/goframe/pkg/app"
	"github.com/polymatx/goframe/pkg/middleware"
	"github.com/polymatx/goframe/pkg/websocket"
)

var hub *websocket.Hub

func main() {
	hub = websocket.NewHub()
	go hub.Run()

	a := app.New(&app.Config{
		Name: "websocket-chat",
		Port: ":8080",
	})

	a.Use(middleware.Recovery())
	a.Use(middleware.Logger())
	a.Use(middleware.DefaultCORS())

	a.Router().HandleFunc("/", serveHome).Methods("GET")
	a.Router().HandleFunc("/ws", handleWebSocket).Methods("GET")
	a.Router().HandleFunc("/broadcast", broadcastMessage).Methods("POST")
	a.Router().HandleFunc("/stats", getStats).Methods("GET")

	fmt.Println("WebSocket chat running on :8080")
	fmt.Println("Open http://localhost:8080 in browser")

	a.StartWithGracefulShutdown()
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user")
	if userID == "" {
		userID = "anonymous"
	}

	if err := hub.Upgrade(w, r, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func broadcastMessage(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)

	var msg struct {
		Message string `json:"message"`
	}

	if err := ctx.Bind(&msg); err != nil {
		ctx.JSONError(400, err)
		return
	}

	hub.Broadcast([]byte(msg.Message))
	ctx.JSON(200, map[string]string{"status": "broadcasted"})
}

func getStats(w http.ResponseWriter, r *http.Request) {
	ctx := app.NewContext(w, r)
	ctx.JSON(200, map[string]interface{}{
		"connections": hub.ConnectionCount(),
	})
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Chat</title>
    <style>
        body { font-family: Arial; max-width: 800px; margin: 50px auto; padding: 20px; }
        #messages { border: 1px solid #ccc; height: 400px; overflow-y: scroll; padding: 10px; margin: 20px 0; }
        .message { padding: 5px; margin: 5px 0; background: #f0f0f0; border-radius: 3px; }
        input { width: 70%; padding: 10px; }
        button { padding: 10px 20px; cursor: pointer; }
    </style>
</head>
<body>
    <h1>WebSocket Chat</h1>
    <div id="messages"></div>
    <input id="messageInput" type="text" placeholder="Type message...">
    <button onclick="sendMessage()">Send</button>
    <p>Connected: <span id="status">No</span> | Users: <span id="users">0</span></p>

    <script>
        let ws;
        const messages = document.getElementById('messages');
        const input = document.getElementById('messageInput');
        const status = document.getElementById('status');

        function connect() {
            ws = new WebSocket('ws://localhost:8080/ws?user=' + Math.random().toString(36).substr(2, 9));
            
            ws.onopen = () => {
                status.textContent = 'Yes';
                status.style.color = 'green';
                updateStats();
            };
            
            ws.onmessage = (event) => {
                const msg = document.createElement('div');
                msg.className = 'message';
                msg.textContent = event.data;
                messages.appendChild(msg);
                messages.scrollTop = messages.scrollHeight;
            };
            
            ws.onclose = () => {
                status.textContent = 'No';
                status.style.color = 'red';
                setTimeout(connect, 3000);
            };
        }

        function sendMessage() {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(input.value);
                input.value = '';
            }
        }

        function updateStats() {
            fetch('/stats')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('users').textContent = data.connections;
                });
        }

        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') sendMessage();
        });

        connect();
        setInterval(updateStats, 5000);
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
