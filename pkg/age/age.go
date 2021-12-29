/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
