package tbcomctl

import (
	tb "gopkg.in/tucnak/telebot.v3"
)

type Form struct {
	ctrls []Controller
	cm    map[string]Controller
}

func NewForm(ctrls ...Controller) *Form {
	if len(ctrls) == 0 {
		panic("creating form with no controllers")
	}
	fm := &Form{
		ctrls: ctrls,
	}
	// name->controller map
	fm.cm = make(map[string]Controller, len(fm.ctrls))

	var prev Controller
	for i, ct := range fm.ctrls {
		var next Controller
		if i < len(fm.ctrls)-1 {
			next = fm.ctrls[i+1]
		}
		ct.SetNext(next)
		ct.SetPrev(prev)
		prev = ct

		if _, exist := fm.cm[ct.Name()]; exist {
			panic("controller " + ct.Name() + " already exist")
		}
		fm.cm[ct.Name()] = ct
		ct.SetForm(fm)
	}
	return fm
}

// SetOverwrite sets the overwrite flag on all controllers within the form.
func (fm *Form) SetOverwrite(b bool) *Form {
	for _, c := range fm.ctrls {
		if p, ok := c.(overwriter); ok {
			p.setOverwrite(b)
		}
	}
	return fm
}

// SetRemoveButtons sets the remove buttons flag on all controllers within the
// form.
func (fm *Form) SetRemoveButtons(b bool) *Form {
	for _, c := range fm.ctrls {
		if p, ok := c.(*Picklist); ok {
			p.removeButtons = b
		}
	}
	return fm
}

func (fm *Form) Handler(c tb.Context) error {
	return fm.ctrls[0].Handler(c)
}

// Controller returns the Form Controller by it's name.
func (fm *Form) Controller(name string) (Controller, bool) {
	c, ok := fm.cm[name]
	return c, ok
}

type onTexter interface {
	OnTextMw(fn tb.HandlerFunc) tb.HandlerFunc
}

// OnTextMiddleware returns the middleware for OnText handler.
func (fm *Form) OnTextMiddleware(onText tb.HandlerFunc) tb.HandlerFunc {
	var mwfn []MiddlewareFunc
	for _, ctrl := range fm.ctrls {
		otmw, ok := ctrl.(onTexter) // if the control satisfies onTexter, it contains middleware function
		if !ok {
			continue
		}
		mwfn = append(mwfn, otmw.OnTextMw)
	}
	return middlewareChain(onText, mwfn...)
}

func middlewareChain(final tb.HandlerFunc, mw ...MiddlewareFunc) tb.HandlerFunc {
	var handler = final
	for i := len(mw) - 1; i >= 0; i-- {
		handler = mw[i](handler)
	}
	return handler
}

// Data returns form data for the recipient.
func (fm *Form) Data(r tb.Recipient) map[string]string {
	data := make(map[string]string, len(fm.ctrls))
	for k, v := range fm.cm {
		val, ok := v.Value(r.Recipient())
		if !ok {
			continue
		}
		data[k] = val
	}
	return data
}

// Value returns the form control value for recipient by name
func (fm *Form) Value(ctrlName, recipient string) (string, bool) {
	ctrl, ok := fm.cm[ctrlName]
	if !ok {
		return "", false
	}
	return ctrl.Value(recipient)
}
