// Package svc is the input of the generr golden test.
package svc

// Service is the type the error functions are generated for.
type Service struct{}

// Default has no annotation and gets the default error function.
func (s *Service) Default() error { return nil }

// Format sets a custom format.
//
// +generr:format=%s: failed to process request: %w
func (s *Service) Format() error { return nil }

// NameSuffix sets a suffix for the generated function name.
//
// +generr:namesuffix=withContext
func (s *Service) NameSuffix() error { return nil }

// Params has parameters but no format, so the format is derived.
//
// +generr:params=string(id);error
func (s *Service) Params(id string) error { return nil }

// ParamsOrder declares the error first; the derived format still ends in it.
//
// +generr:params=error;string(id);int
func (s *Service) ParamsOrder(id string) error { return nil }

// ParamsNoError has no error parameter at all.
//
// +generr:params=string(id)
func (s *Service) ParamsNoError(id string) error { return nil }

// FormatAndParams combines all options.
//
// +generr:format=%s(owner:'%s'/after:'%s'): %w,params=string(ownerID);string(afterID);error,namesuffix=forOwner
func (s *Service) FormatAndParams() error { return nil }

// Quoted uses a quoted format, which may contain commas and equal signs.
//
// +generr:format="%s: id=%d, retries=%d: %w",params=int(id);int(retries);error
func (s *Service) Quoted() error { return nil }

// Multiple generates several error functions.
//
// +generr
// +generr:namesuffix=NotFound,format=%s: item not found: %w
// +generr:format=%s: invalid %v: %w,params=[]string(ids);error
func (s *Service) Multiple() error { return nil }

// Skipped is ignored.
//
// +generr:skip
func (s *Service) Skipped() error { return nil }

func (s *Service) unexported() error { return nil }
