//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package metering

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

func TestMeteringToolFlagConversion(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name           string
		interval       kubermaticv1.Interval
		intervalInDays int
		expectedResult string
	}{
		{
			name:           "daily interval",
			interval:       kubermaticv1.Day,
			intervalInDays: 0,
			expectedResult: Yesterday,
		},
		{
			name:           "weekly interval",
			interval:       kubermaticv1.Week,
			intervalInDays: 0,
			expectedResult: LastWeek,
		},
		{
			name:           "monthly interval",
			interval:       kubermaticv1.Month,
			intervalInDays: 0,
			expectedResult: LastMonth,
		},
		{
			name:           "n-days interval",
			interval:       kubermaticv1.Month,
			intervalInDays: 42,
			expectedResult: LastNumberOfDays + "42",
		},
	}
	for _, tc := range testcases {
		actualResult := intervalToFlag(tc.intervalInDays, tc.interval)
		if actualResult != tc.expectedResult {
			t.Fatalf("expected %s got %s", tc.expectedResult, actualResult)
		}
	}
}
