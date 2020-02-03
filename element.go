package rod

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/cdp"
)

// Element represents the DOM element
type Element struct {
	ctx  context.Context
	page *Page

	ObjectID string

	timeoutCancel func()
}

// Ctx sets the context for later operation
func (el *Element) Ctx(ctx context.Context) *Element {
	newObj := *el
	newObj.ctx = ctx
	return &newObj
}

// Timeout sets the timeout for later operation
func (el *Element) Timeout(d time.Duration) *Element {
	ctx, cancel := context.WithTimeout(el.ctx, d)
	el.timeoutCancel = cancel
	return el.Ctx(ctx)
}

// CancelTimeout ...
func (el *Element) CancelTimeout() {
	if el.timeoutCancel != nil {
		el.timeoutCancel()
	}
}

func (el *Element) describe() (kit.JSONResult, error) {
	node, err := el.page.Call(el.ctx,
		"DOM.describeNode",
		cdp.Object{
			"objectId": el.ObjectID,
		},
	)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// FrameE ...
func (el *Element) FrameE() (*Page, error) {
	node, err := el.describe()
	if err != nil {
		return nil, err
	}

	newPage := *el.page
	newPage.FrameID = node.Get("node.frameId").String()
	newPage.element = el

	return &newPage, newPage.initIsolatedWorld()
}

// Frame creates a page instance that represents the iframe
func (el *Element) Frame() *Page {
	f, err := el.FrameE()
	kit.E(err)
	return f
}

// FocusE ...
func (el *Element) FocusE() error {
	err := el.ScrollIntoViewIfNeededE()
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, `() => this.focus()`)
	return err
}

// Focus sets focus on the specified element
func (el *Element) Focus() {
	kit.E(el.FocusE())
}

// ScrollIntoViewIfNeededE ...
func (el *Element) ScrollIntoViewIfNeededE() error {
	_, err := el.EvalE(true, `async () => {
		if (!this.isConnected)
			return 'Node is detached from document';
		if (this.nodeType !== Node.ELEMENT_NODE)
			return 'Node is not of type HTMLElement';
	
		const visibleRatio = await new Promise(resolve => {
			const observer = new IntersectionObserver(entries => {
				resolve(entries[0].intersectionRatio);
				observer.disconnect();
			});
			observer.observe(this);
		});
		if (visibleRatio !== 1.0)
			this.scrollIntoView({ block: 'center', inline: 'center', behavior: 'instant' });
		return false;
	}`)
	return err
}

// ScrollIntoViewIfNeeded scrolls the current element into the visible area of the browser
// window if it's not already within the visible area.
func (el *Element) ScrollIntoViewIfNeeded() {
	kit.E(el.ScrollIntoViewIfNeededE())
}

// ClickE ...
func (el *Element) ClickE(button string) error {
	err := el.ScrollIntoViewIfNeededE()
	if err != nil {
		return err
	}

	box, err := el.BoxE()
	if err != nil {
		return err
	}

	x := box.Get("left").Int() + box.Get("width").Int()/2
	y := box.Get("top").Int() + box.Get("height").Int()/2

	err = el.page.Mouse.MoveToE(x, y)
	if err != nil {
		return err
	}

	defer el.trace(button + " click")()

	return el.page.Mouse.ClickE(button)
}

// Click the element
func (el *Element) Click() {
	kit.E(el.ClickE("left"))
}

// PressE ...
func (el *Element) PressE(key rune) error {
	err := el.FocusE()
	if err != nil {
		return err
	}

	defer el.trace("press " + string(key))()

	return el.page.Keyboard.PressE(key)
}

// Press a key
func (el *Element) Press(key rune) {
	kit.E(el.PressE(key))
}

// InputE ...
func (el *Element) InputE(text string) error {
	err := el.FocusE()
	if err != nil {
		return err
	}

	defer el.trace("input " + text)()

	err = el.page.Keyboard.InsertTextE(text)
	if err != nil {
		return err
	}

	_, err = el.EvalE(true, `() => {
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
	}`)
	return err
}

// Input wll click the element and input the text
func (el *Element) Input(text string) {
	kit.E(el.InputE(text))
}

// SelectE ...
func (el *Element) SelectE(selectors ...string) error {
	defer el.trace(fmt.Sprintf(
		`<span style="color: #777;">select</span> <code>%s</code>`,
		strings.Join(selectors, "; ")))()
	el.page.browser.slowmotion("Input.select")

	_, err := el.EvalE(true, `selectors => {
		selectors.forEach(s => {
			Array.from(this.options).forEach(el => {
				try {
					if (el.innerText === s || el.matches(s)) {
						el.selected = true
					}
				} catch {}
			})
		})
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
	}`, selectors)
	return err
}

// Select the option elements that match the selectors, the selector can be text content or css selector
func (el *Element) Select(selectors ...string) {
	kit.E(el.SelectE(selectors...))
}

// SetFilesE ...
func (el *Element) SetFilesE(paths []string) error {
	absPaths := []string{}
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		absPaths = append(absPaths, absPath)
	}

	_, err := el.page.Call(el.ctx, "DOM.setFileInputFiles", cdp.Object{
		"files":    absPaths,
		"objectId": el.ObjectID,
	})
	return err
}

// SetFiles sets files for the given file input element
func (el *Element) SetFiles(paths ...string) {
	kit.E(el.SetFilesE(paths))
}

// TextE ...
func (el *Element) TextE() (string, error) {
	str, err := el.EvalE(true, `() => this.innerText`)
	return str.String(), err
}

// Text gets the innerText of the element
func (el *Element) Text() string {
	s, err := el.TextE()
	kit.E(err)
	return s
}

// HTMLE ...
func (el *Element) HTMLE() (string, error) {
	str, err := el.EvalE(true, `() => this.outerHTML`)
	return str.String(), err
}

// HTML gets the outerHTML of the element
func (el *Element) HTML() string {
	s, err := el.HTMLE()
	kit.E(err)
	return s
}

// WaitE ...
func (el *Element) WaitE(js string, params ...interface{}) error {
	return cdp.Retry(el.ctx, func() error {
		res, err := el.EvalE(true, js, params...)
		if err != nil {
			return err
		}

		if res.Bool() {
			return nil
		}

		return cdp.ErrNotYet
	})
}

// Wait until the js returns true
func (el *Element) Wait(js string, params ...interface{}) {
	kit.E(el.WaitE(js, params))
}

// WaitVisibleE ...
func (el *Element) WaitVisibleE() error {
	return el.WaitE(`() => {
		var box = this.getBoundingClientRect()
		var style = window.getComputedStyle(this)
		return style.display != 'none' &&
			style.visibility != 'hidden' &&
			!!(box.top || box.bottom || box.width || box.height)
	}`)
}

// WaitVisible until the element is visible
func (el *Element) WaitVisible() {
	kit.E(el.WaitVisibleE())
}

// WaitInvisibleE ...
func (el *Element) WaitInvisibleE() error {
	return el.WaitE(`() => {
		var box = this.getBoundingClientRect()
		return window.getComputedStyle(this).visibility == 'hidden' ||
			!(box.top || box.bottom || box.width || box.height)
	}`)
}

// WaitInvisible until the element is not visible or removed
func (el *Element) WaitInvisible() {
	kit.E(el.WaitInvisibleE())
}

// BoxE ...
func (el *Element) BoxE() (kit.JSONResult, error) {
	box, err := el.EvalE(true, `() => {
		var box = this.getBoundingClientRect().toJSON()
		if (this.tagName === 'IFRAME') {
			var style = window.getComputedStyle(this)
			box.left += parseInt(style.paddingLeft) + parseInt(style.borderLeftWidth)
			box.top += parseInt(style.paddingTop) + parseInt(style.borderTopWidth)
		}
		return box
	}`)
	if err != nil {
		return nil, err
	}

	var j map[string]interface{}
	kit.E(json.Unmarshal([]byte(box.String()), &j))

	if el.page.isIframe() {
		frameRect, err := el.page.element.BoxE() // recursively get the box
		if err != nil {
			return nil, err
		}
		j["left"] = box.Get("left").Int() + frameRect.Get("left").Int()
		j["top"] = box.Get("top").Int() + frameRect.Get("top").Int()
	}
	return kit.JSON(kit.MustToJSON(j)), nil
}

// Box returns the size of an element and its position relative to the main frame.
// It will recursively calculate the box with all ancestors. The spec is here:
// https://developer.mozilla.org/en-US/docs/Web/API/Element/getBoundingClientRect
func (el *Element) Box() kit.JSONResult {
	box, err := el.BoxE()
	kit.E(err)
	return box
}

// ElementE ...
func (el *Element) ElementE(selector string) (*Element, error) {
	return el.ElementByJSE(`s => this.querySelector(s)`, selector)
}

// Element waits and returns the first element in the page that matches the selector
func (el *Element) Element(selector string) *Element {
	el, err := el.ElementE(selector)
	kit.E(err)
	return el
}

// ElementByJSE ...
func (el *Element) ElementByJSE(js string, params ...interface{}) (*Element, error) {
	return el.page.ElementByJSE(el.ObjectID, js, params)
}

// ElementByJS waits and returns the element from the return value of the js
func (el *Element) ElementByJS(js string, params ...interface{}) *Element {
	el, err := el.ElementByJSE(js, params...)
	kit.E(err)
	return el
}

// ElementsE ...
func (el *Element) ElementsE(selector string) ([]*Element, error) {
	return el.ElementsByJSE(`s => this.querySelectorAll(s)`, selector)
}

// Elements returns all elements that match the selector
func (el *Element) Elements(selector string) []*Element {
	list, err := el.ElementsE(selector)
	kit.E(err)
	return list
}

// ElementsByJSE ...
func (el *Element) ElementsByJSE(js string, params ...interface{}) ([]*Element, error) {
	return el.page.ElementsByJSE(el.ObjectID, js, params)
}

// ElementsByJS returns the elements from the return value of the js
func (el *Element) ElementsByJS(js string, params ...interface{}) []*Element {
	list, err := el.ElementsByJSE(js, params...)
	kit.E(err)
	return list
}

// ParentE ...
func (el *Element) ParentE() (*Element, error) {
	return el.ElementByJSE(`() => this.parentElement`)
}

// Parent returns the parent element
func (el *Element) Parent() *Element {
	parent, err := el.ParentE()
	kit.E(err)
	return parent
}

// NextE ...
func (el *Element) NextE() (*Element, error) {
	return el.ElementByJSE(`() => this.nextElementSibling`)
}

// Next returns the next sibling element
func (el *Element) Next() *Element {
	parent, err := el.NextE()
	kit.E(err)
	return parent
}

// PreviousE ...
func (el *Element) PreviousE() (*Element, error) {
	return el.ElementByJSE(`() => this.previousElementSibling`)
}

// Previous returns the previous sibling element
func (el *Element) Previous() *Element {
	parent, err := el.PreviousE()
	kit.E(err)
	return parent
}

// EvalE ...
func (el *Element) EvalE(byValue bool, js string, params ...interface{}) (kit.JSONResult, error) {
	return el.page.EvalE(byValue, el.ObjectID, js, params)
}

// Eval evaluates js function on the element, the first param must be a js function definition
// For example: el.Eval(`name => this.getAttribute(name)`, "value")
func (el *Element) Eval(js string, params ...interface{}) kit.JSONResult {
	res, err := el.EvalE(true, js, params...)
	kit.E(err)
	return res
}
