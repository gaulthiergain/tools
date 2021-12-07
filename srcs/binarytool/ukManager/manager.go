package ukManager

type Manager struct {
	Unikernels *Unikernels
	MicroLibs  map[string]MicroLibs
}

type MicroLibs struct {
	Name      string
	StartAddr uint64
	Size      uint64
	instance  int
}

func checkInstance() {

}
