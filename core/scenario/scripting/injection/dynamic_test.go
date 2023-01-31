package injection

import (
	"testing"
)

func TestDynamicVariableRace(t *testing.T) {
	num := 10
	ei := EnvironmentInjector{}
	for key := range dynamicFakeDataMap {
		for i := 0; i < num; i++ {
			go ei.getFakeData(key)
		}
	}
}
