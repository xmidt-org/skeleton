// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package apiauth

import (
	"fmt"
	"reflect"
)

// Option is an interface that is used to apply options to the Auth struct.
type Option interface {
	apply(*Auth) error
}

// optionFunc is a function that is used to apply options to the Auth struct.
type optionFunc func(*Auth) error

func (of optionFunc) apply(a *Auth) error {
	return of(a)
}

func WithConfig(c Config) optionFunc {
	return func(a *Auth) error {
		a.config = c
		return nil
	}
}

//------------------------------------------------------------------------------

func validate() optionFunc {
	return func(a *Auth) error {
		if a.config.Disable {
			expect := Config{
				Disable: true,
			}

			if !reflect.DeepEqual(a.config, expect) {
				return fmt.Errorf("%w: disable cannot have additional values", ErrInvalidConfig)
			}
		}

		if reflect.DeepEqual(a.config, Config{}) {
			return fmt.Errorf("%w: empty configuration is not valid, set 'disable' to true if no validation is wanted", ErrInvalidConfig)
		}

		return nil
	}
}
