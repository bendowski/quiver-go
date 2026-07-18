package testpkg

import "fmt"

// Greeter can greet.
type Greeter struct {
	Name string
	Age  int
}

// Hello greets someone.
func (g *Greeter) Hello() string {
	return fmt.Sprintf("Hello, %s!", g.Name)
}

// Greet is an exported top-level function.
func Greet(name string) string {
	return fmt.Sprintf("Hi, %s!", name)
}

// hidden is unexported.
func hidden() {}

const DefaultName = "World"

var greeting = "Hello"
