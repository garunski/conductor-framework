package errors

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalid            = errors.New("invalid")
	ErrInvalidYAML        = errors.New("invalid yaml")
	ErrStorage            = errors.New("storage error")
	ErrKubernetes         = errors.New("kubernetes error")
	ErrReconciliation     = errors.New("reconciliation error")
	ErrEventStore         = errors.New("event store error")
	ErrMissingParameter   = errors.New("missing parameter")
	ErrInvalidParameter   = errors.New("invalid parameter")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidNamespace   = errors.New("invalid namespace")
	ErrInvalidServiceName = errors.New("invalid service name")
)

