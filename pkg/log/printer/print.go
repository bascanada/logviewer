package printer

import (
	"context"
	"os"

	"github.com/bascanada/logviewer/pkg/log/client"
)

type PrintPrinter struct{}

func (pp PrintPrinter) Display(ctx context.Context, result client.LogSearchResult) (bool, error) {

	return WrapIoWritter(ctx, result, os.Stdout, func() {})
}
