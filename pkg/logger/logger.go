// Package logger provides context-aware structured logging.
// This is a compatibility wrapper for the xlog package.
package logger

import (
	"github.com/polymatx/goframe/pkg/xlog"
)

// Re-export all xlog functions for backward compatibility
var (
	Initialize       = xlog.Initialize
	Get              = xlog.Get
	GetWithError     = xlog.GetWithError
	GetWithField     = xlog.GetWithField
	GetWithFields    = xlog.GetWithFields
	SetField         = xlog.SetField
	SetFields        = xlog.SetFields
	GetSpecialLogger = xlog.GetSpecialLogger
	SetLogLocation   = xlog.SetLogLocation
)
