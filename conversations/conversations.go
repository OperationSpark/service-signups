package conversations

import "context"

type Service struct {
}

func (s Service) Run(ctx context.Context, signupID string, conversationID string) error {
	return nil
}
