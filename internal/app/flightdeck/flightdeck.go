package flightdeck

import "context"

type Deployer interface {
	Deploy(context.Context, string) error
}
