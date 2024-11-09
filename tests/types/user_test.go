package types

import (
	"testing"
	"time"

	"github.com/nbittich/wtm/types"
)

func TestIsAvailable(t *testing.T) {
	tests := []struct {
		label        string
		start        time.Time
		end          time.Time
		availability types.UserNormalAvailability
		expectedErr  error
		expectedRes  bool
	}{
		{
			label: "Monday November 4th, 23:00h->05:00h next day",
			availability: types.UserNormalAvailability{
				Days:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Tuesday, time.Thursday, time.Friday, time.Saturday},
				MinHour:     22,
				MaxHour:     6,
				HoursPerDay: 8,
			},

			start:       time.Date(2024, time.November, 4, 23, 0, 0, 0, time.Now().Location()),
			end:         time.Date(2024, time.November, 5, 4, 0, 0, 0, time.Now().Location()),
			expectedErr: nil,
			expectedRes: true,
		},

		{
			label: "Monday November 4th, 06:00h->14:00h",
			availability: types.UserNormalAvailability{
				Days:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Tuesday, time.Thursday, time.Friday, time.Saturday},
				MinHour:     6,
				MaxHour:     22,
				HoursPerDay: 8,
			},

			start:       time.Date(2024, time.November, 4, 6, 0, 0, 0, time.Now().Location()),
			end:         time.Date(2024, time.November, 4, 14, 0, 0, 0, time.Now().Location()),
			expectedErr: nil,
			expectedRes: true,
		},
		{
			label: "Monday November 4th, 14:00h->22:00h",
			availability: types.UserNormalAvailability{
				Days:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Tuesday, time.Thursday, time.Friday, time.Saturday},
				MinHour:     6,
				MaxHour:     22,
				HoursPerDay: 8,
			},

			start:       time.Date(2024, time.November, 4, 14, 0, 0, 0, time.Now().Location()),
			end:         time.Date(2024, time.November, 4, 22, 0, 0, 0, time.Now().Location()),
			expectedErr: nil,
			expectedRes: true,
		},
		{
			label: "Monday November 4th, 22:00h->04:00h the next day",
			availability: types.UserNormalAvailability{
				Days:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Tuesday, time.Thursday, time.Friday, time.Saturday},
				MinHour:     22,
				MaxHour:     6,
				HoursPerDay: 8,
			},

			start:       time.Date(2024, time.November, 4, 22, 0, 0, 0, time.Now().Location()),
			end:         time.Date(2024, time.November, 5, 4, 0, 0, 0, time.Now().Location()),
			expectedErr: nil,
			expectedRes: true,
		},

		{
			label: "Monday November 4th, 22:00h->07:00h the next day (9h)",
			availability: types.UserNormalAvailability{
				Days:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Tuesday, time.Thursday, time.Friday, time.Saturday},
				MinHour:     22,
				MaxHour:     6,
				HoursPerDay: 8,
			},

			start:       time.Date(2024, time.November, 4, 22, 0, 0, 0, time.Now().Location()),
			end:         time.Date(2024, time.November, 5, 7, 0, 0, 0, time.Now().Location()),
			expectedErr: nil,
			expectedRes: false,
		},
		{
			label: "Monday November 4th, 22:00h->04:00h the next day but minHour 6 and maxHour 8",
			availability: types.UserNormalAvailability{
				Days:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Tuesday, time.Thursday, time.Friday, time.Saturday},
				MinHour:     6,
				MaxHour:     8,
				HoursPerDay: 8,
			},

			start:       time.Date(2024, time.November, 4, 22, 0, 0, 0, time.Now().Location()),
			end:         time.Date(2024, time.November, 5, 4, 0, 0, 0, time.Now().Location()),
			expectedErr: nil,
			expectedRes: false,
		},
	}
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			res, err := test.availability.IsAvailable(test.start.Format(types.BelgianDateTimeFormat), test.end.Format(types.BelgianDateTimeFormat))
			if err != test.expectedErr {
				t.Fatal(err)
			}
			if res != test.expectedRes {
				t.Errorf("%t!=%t: expect start: %s : end: %s  =>  availability: %v+", res, test.expectedRes,
					test.start.Format(types.BelgianDateTimeFormat), test.end.Format(types.BelgianDateTimeFormat),
					test.availability)
			}
		})
	}
}
