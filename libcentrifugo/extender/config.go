package extender

import "fmt"

type Config struct {
	AggrNode    string
	AggrPort    string
	ServiceName string
	ACL         string
}

func validate(c Config) error {
	if len(c.AggrNode) == 0 {
		return fmt.Errorf("Empty AggrNode")
	}
	if len(c.AggrPort) == 0 {
		return fmt.Errorf("Empty AggrPort")
	}
	if len(c.ServiceName) == 0 {
		return fmt.Errorf("Empty ServiceName")
	}
	// ACL may be empty
	return nil
}
