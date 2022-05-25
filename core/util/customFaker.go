package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	maxUint = ^uint(0)
	minUint = 0
	maxInt  = int(maxUint >> 1)
	minInt  = -maxInt - 1
)

var (
	JobDescriptors = []string{
		"Lead", "Senior", "Direct", "Corporate", "Dynamic", "Future", "Product", "National", "Regional", "District",
		"Central", "Global", "Customer", "Investor", "Dynamic", "International", "Legacy", "Forward", "Internal", "Human",
		"Chief", "Principal"}
	JobAreas = []string{
		"Solutions", "Program", "Brand", "Security", "Research", "Marketing", "Directives", "Implementation",
		"Integration", "Functionality", "Response", "Paradigm", "Tactics", "Identity", "Markets", "Group", "Division",
		"Applications", "Optimization", "Operations", "Infrastructure", "Intranet", "Communications", "Web", "Branding",
		"Quality", "Assurance", "Mobility", "Accounts", "Data", "Creative", "Configuration", "Accountability", "Interactions",
		"Factors", "Usability", "Metrics"}
	Jobs = []string{
		"Supervisor", "Associate", "Executive", "Liaison", "Officer", "Manager", "Engineer", "Specialist", "Director",
		"Coordinator", "Administrator", "Architect", "Analyst", "Designer", "Planner", "Orchestrator", "Technician",
		"Developer", "Producer", "Consultant", "Assistant", "Facilitator", "Agent", "Representative", "Strategist"}
)

type CustomFaker struct {
	Generator *rand.Rand
}

// RandomDigit returns a fake random digit for Faker
func (f CustomFaker) RandomDigit() int {
	return f.Generator.Int() % 10
}

func (f CustomFaker) RandomFloat() float64 {
	return f.Generator.Float64()
}

func (f CustomFaker) RandomString() string {
	return f.RandomStringWithLength(10)
}

func (f CustomFaker) RandomStringWithLength(l int) string {
	r := []string{}
	for i := 0; i < l; i++ {
		r = append(r, f.RandomLetter())
	}
	return strings.Join(r, "")
}

func (f CustomFaker) RandomLetter() string {
	return fmt.Sprintf("%c", f.IntBetween(97, 122))
}

func (f CustomFaker) IntBetween(min, max int) int {
	diff := max - min

	if diff < 0 {
		diff = 0
	}

	if diff == 0 {
		return min
	}

	if diff == maxInt {
		return f.Generator.Intn(diff)
	}

	return f.Generator.Intn(diff+1) + min
}

// New returns a new instance of Faker instance with a random seed
func NewCustomFaker() (f CustomFaker) {
	seed := rand.NewSource(time.Now().Unix())
	f = NewWithSeed(seed)
	return
}

// NewWithSeed returns a new instance of Faker instance with a given seed
func NewWithSeed(src rand.Source) (f CustomFaker) {
	generator := rand.New(src)
	f = CustomFaker{Generator: generator}
	return
}
