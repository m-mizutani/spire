package infra

type Clients struct {
}

type Option func(c *Clients)

func New(options ...Option) *Clients {
	clients := &Clients{}

	for _, opt := range options {
		opt(clients)
	}

	return clients
}
