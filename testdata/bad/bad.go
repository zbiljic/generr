// Package bad is the input of the generr test for rejected annotations.
package bad

// Service is the type the error functions are generated for.
type Service struct{}

// Broken has a format that does not consume the operation name.
//
// +generr:format=failed to process request: %w
func (s *Service) Broken() error { return nil }

// Collide has two methods whose error functions would get the same name.
type Collide struct{}

// Get would get errGet.
func (c *Collide) Get() error { return nil }

func (c *Collide) get() error { return nil }

// Unimported refers to a package that no file of this package imports.
type Unimported struct{}

// Wait needs the time package.
//
// +generr:params=time.Duration(d);error
func (u *Unimported) Wait() error { return nil }
