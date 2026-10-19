package server

import "testing"

// .   F3a   assertKoreanError( ·  0)
// . SetWorkers(n<=0)  m.pg.SetSetting     DB 
//
//   (errWorkersPositive   ):    PUT 
// (server.go:3512 → writeErr 400)   ,  (actool)
func TestSetWorkersErrorLocalized(t *testing.T) {
	err := (&Manager{}).SetWorkers(0)
	if err == nil {
		t.Fatal("   0   ")
	}
	assertKoreanError(t, "set_workers_nonpositive", err.Error())
}
