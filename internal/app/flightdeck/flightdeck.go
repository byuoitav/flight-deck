package flightdeck

import "context"

type Deployer interface {
	Deploy(context.Context, string) error
	Refloat(context.Context, string) error
	Rebuild(context.Context, string) error
}
