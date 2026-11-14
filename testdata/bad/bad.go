// Package bad is the input of the generr test for rejected annotations.
package bad

// Service is the type the error functions are generated for.
type Service struct{}

// Broken has a format that does not consume the operation name.
//
// +generr:format=failed to process request: %w
func (s *Service) Broken() error { return nil }
