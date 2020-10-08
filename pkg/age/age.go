// ------------------------------------------------------------
// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
// ------------------------------------------------------------

package age

import (
	"fmt"
	"time"
)

func GetAge(t time.Time) string {
	d := time.Since(t)
	if d.Seconds() <= 60 {
		return fmt.Sprintf("%vs", int(d.Seconds()))
	} else if d.Minutes() <= 60 {
		return fmt.Sprintf("%vm", int(d.Minutes()))
	} else if d.Hours() <= 24 {
		return fmt.Sprintf("%vh", int(d.Hours()))
	} else if d.Hours() > 24 {
		return fmt.Sprintf("%vd", int(d.Hours()/24))
	}

	return ""
}
