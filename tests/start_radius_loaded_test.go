package tests

import (
	"fmt"
	engine "settlers2-final-go/InternalMapManipulation"
	"testing"
)

func TestStartRadiusWithFullDefaultSettings(t *testing.T) {
	for _, players := range []int{3, 5, 6} {
		for _, rad := range []int{10, 90} {
			t.Run(fmt.Sprintf("%d-%d", players, rad), func(t *testing.T) {
				_, err := engine.Generate(buildRotationPayload(players, 109, rad, false))
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
