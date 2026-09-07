package internal_sip_telephony

import "testing"

func TestLoadRingtoneBytes_ExactNames(t *testing.T) {
	names := []string{"ringtone_us", "ringtone_uk", "ringtone_in"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if len(LoadRingtoneBytes(name)) == 0 {
				t.Fatalf("expected ringtone bytes for %s", name)
			}
		})
	}
}
