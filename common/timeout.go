package common

import "time"

// TimeoutSettings holds information on timeout settings.
type TimeoutSettings struct {
	parent                   *TimeoutSettings
	defaultTimeout           *int64
	defaultNavigationTimeout *time.Duration
}

// NewTimeoutSettings creates a new timeout settings object.
func NewTimeoutSettings(parent *TimeoutSettings) *TimeoutSettings {
	t := &TimeoutSettings{
		parent:                   parent,
		defaultTimeout:           nil,
		defaultNavigationTimeout: nil,
	}
	return t
}

func (t *TimeoutSettings) setDefaultTimeout(timeout int64) {
	t.defaultTimeout = &timeout
}

func (t *TimeoutSettings) setDefaultNavigationTimeout(timeout time.Duration) {
	t.defaultNavigationTimeout = &timeout
}

func (t *TimeoutSettings) navigationTimeout() time.Duration {
	if t.defaultNavigationTimeout != nil {
		return *t.defaultNavigationTimeout
	}
	if t.defaultTimeout != nil {
		return time.Duration(*t.defaultTimeout * time.Hour.Milliseconds())
	}
	if t.parent != nil {
		return t.parent.navigationTimeout()
	}
	return DefaultTimeout
}

func (t *TimeoutSettings) timeout() int64 {
	if t.defaultTimeout != nil {
		return *t.defaultTimeout
	}
	if t.parent != nil {
		return t.parent.timeout()
	}
	return int64(DefaultTimeout.Milliseconds())
}
