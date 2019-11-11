// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package protocols

import (
	"errors"
	multierr "github.com/hashicorp/go-multierror"
	"strings"
)

func MultiErrorBasicFormat(errors []error) string {
	var ret string

	for _, e := range errors {
		ret += e.Error() + "\n\n"
	}

	return strings.TrimRight(ret, "\n")
}

func NewMultiError() *multierr.Error {
	var err multierr.Error

	err.ErrorFormat = MultiErrorBasicFormat
	return &err
}

func PrefixError(e error, prefix string) error {
	if e == nil {
		return nil
	}

	return errors.New(prefix + e.Error())
}

func AddErrorContext(e error, ctx string) error {
	return PrefixError(e, "["+ctx+"]\n")
}
