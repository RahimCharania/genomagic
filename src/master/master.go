// master is responsible for interacting with the user to take in the contigs that need to be assembled. Files
// that are specified by the user are passed to a slave that knows how to schedule Docker containers for all the
// assemblers that are integrated with genomagic
package master

import "github.com/genomagic/src/slave"

// mst defines the master struct, which is used to coordinate slaves and launch assembly, parsing, and
// reporting slaves
type mst struct {
	fileNames       []string          // a slice of file names that contain contigs to be assembled and analyzed
	assemblyResults chan slave.Result // a collection of assembly results used by the assembly slave
	parsingResults  chan slave.Result // a collection of parsing results used by the parsing slave
}

// Process launches the assembly of the contings it was created with
func (m *mst) Process() {
}
