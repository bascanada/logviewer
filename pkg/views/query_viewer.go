package views

import (
	"context"
	"errors"
	"log"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/client/config"
	"github.com/berlingoqc/logviewer/pkg/log/factory"
	"github.com/berlingoqc/logviewer/pkg/log/printer"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	notSelectedColor = tcell.ColorGrey
	selectedColor    = tcell.ColorGreen
)

type tviewWrapper struct {
	app    *tview.Application
	parent *tview.Flex
	tv     *tview.TextView
	fields *tview.DeepList
	result client.LogSearchResult
}

func (tv tviewWrapper) Display(ctx context.Context, result client.LogSearchResult) error {
	go printer.WrapIoWritter(ctx, result, tv.tv, func() {
		// TODO: scroll to end if we are not scroll up
		tv.app.QueueUpdateDraw(func() {
			tv.tv.ScrollToEnd()
		})
	})

	return nil
}

func createLogTextView(app *tview.Application, name string) *tviewWrapper {
	parentFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

	tv := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetScrollable(true)

	tv.SetBorder(true)
	tv.SetBorderColor(notSelectedColor)
	tv.SetTitle(name)

	parentFlex.AddItem(tv, 0, 100, false)

	wrapper := new(tviewWrapper)
	wrapper.app = app
	wrapper.tv = tv
	wrapper.parent = parentFlex

	parentFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlB {
			if wrapper.fields == nil {
				fields, _, err := wrapper.result.GetFields()
				if err != nil {
					log.Println(err.Error())
					return event
				}

				listPrimitive := tview.NewDeepList()

				i := 0
				for k, values := range fields {
					ii := i
					listPrimitive.AddItem(k, "", rune(0), func() {
						listPrimitive.ToggleSubListDisplay(ii)
					})
					for _, v := range values {
						listPrimitive.AddSubItem(v, "", rune(0), false, nil)
					}
					i += 1
				}

				wrapper.fields = listPrimitive
				parentFlex.AddItem(wrapper.fields, 0, 40, true)
			} else {
				wrapper.parent.RemoveItem(wrapper.fields)
				wrapper.fields = nil
			}
			return nil
		}
		return event
	})

	return wrapper
}

// Return the queryBox to display one output of logs
func getQueryBox(app *tview.Application, searchesId []string) (*tview.Flex, map[string]*tviewWrapper, error) {

	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	tviewWrappers := make(map[string]*tviewWrapper)

	elementProportion := 100 / len(searchesId)

	for _, v := range searchesId {
		wrapper := createLogTextView(app, v)
		flex.AddItem(wrapper.parent, 0, elementProportion, false)
		tviewWrappers[v] = wrapper
	}

	return flex, tviewWrappers, nil
}

func RunQueryViewApp(config config.ContextConfig, searchIds []string) error {

	app := tview.NewApplication().EnableMouse(true)

	clientFactory, err := factory.GetLogClientFactory(config.Clients)
	if err != nil {
		return err
	}

	if len(searchIds) == 0 {
		return errors.New("required multiple searches for query")
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, config)
	if err != nil {
		return err
	}

	grid, wrappers, err := getQueryBox(app, searchIds)
	if err != nil {
		return err
	}

	ctx := context.Background()
	ctx, _ = context.WithCancel(ctx)
	//defer cancel()

	for k, v := range wrappers {
		result, err := searchFactory.GetSearchResult(ctx, k, []string{}, client.LogSearch{})
		v.result = result
		if err != nil {
			return err
		}

		err = v.Display(ctx, result)
		if err != nil {
			panic(err)
		}
	}

	if err := app.SetRoot(grid, true).Run(); err != nil {
		return err
	}

	return nil
}
