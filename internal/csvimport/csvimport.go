package csvimport

import (
	"context"
	"fmt"
	"github.com/andresbott/etna/internal/accounting"
)

// takes a csv, processes it as per notes and returns a list of transcations
// optional add hooks or AI mechanism to map items
func Process(ctx context.Context, store *accounting.Store, file string) error {

}

// I need a set of rules
// import profile contains N amount of regex (?) or column name to map the column to the target field
// category profiles contain regex to map descriptions to categories
