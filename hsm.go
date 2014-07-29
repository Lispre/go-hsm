package hsm

import "container/list"

type HSMType uint32

// The types of HSM.
const (
    HSMTypeStd HSMType = iota
    HSMTypeUser
)

// HSM represents the interface that every state machine class
// should implement.
type HSM interface {
    // Returns the type of this hsm
    Type() HSMType

    // Runs the initialization of this hsm
    Init()
    // Dispatch event to state machine
    Dispatch(event Event)

    // Returns current state of this hsm
    GetState() State
    // Tests whether this hsm is in specified state. It works no matter
    // stateID is in any level as a parent state of current state.
    IsIn(stateID string) bool

    // Transfer to specified target state during state intialization.
    QInit(targetStateID string)
    // Transfer to specified target state as normal state transfer.
    QTran(targetStateID string)
    // Transfer to specified target state as normal state transfer,
    // along with specified event dispatched during transfer procedure.
    QTranOnEvent(targetStateID string, event Event)
}

// StdHSM is the default HSM implementation.
// Any HSM derived could reuse it as anonymous field.
type StdHSM struct {
    // The type of concrete HSM
    MyType HSMType
    // The state that handles event(it could Super(), Super().Super(), ...)
    // of current state
    SourceState State
    // The current state(it could be child, child's child of SourceState)
    State State
    // The global map for all states and their names in this state machine
    StateTable map[string]State
}

// Constructor for StdHSM. The initial must set top as parent state.
func NewStdHSM(myType HSMType, top, initial State) *StdHSM {
    AssertEqual(TopStateID, top.ID())
    AssertEqual(InitialStateID, initial.ID())
    hsm := &StdHSM{
        MyType:      myType,
        SourceState: initial,
        State:       top,
        StateTable:  make(map[string]State),
    }
    hsm.StateTable[top.ID()] = top
    // setup state table
    hsm.setupStateTable()
    return hsm
}

func (self *StdHSM) Type() HSMType {
    return self.MyType
}

// setupStateTable() initializes StateTable properly
// with all states and their names.
func (self *StdHSM) setupStateTable() {
    for traverse_queue := self.State.Children(); len(traverse_queue) != 0; {
        state := traverse_queue[0]
        traverse_queue = traverse_queue[1:]
        _, ok := self.StateTable[state.ID()]
        AssertFalse(ok)
        self.StateTable[state.ID()] = state
        children := state.Children()
        for _, state := range children {
            traverse_queue = append(traverse_queue, state)
        }
    }
}

// Init() is part of interface HSM.
func (self *StdHSM) Init() {
    self.Init2(self, StdEvents[EventInit])
}

// Init2() is a helper function to initialize the whole state machine.
// All state initialization actions started from initial state
// would be triggered.
func (self *StdHSM) Init2(hsm HSM, event Event) {
    // health check
    AssertNotEqual(nil, self.State)
    AssertNotEqual(nil, self.SourceState)
    // check HSM is not executed yet
    AssertEqual(self.StateTable[TopStateID], self.State)
    AssertEqual(self.StateTable[InitialStateID], self.SourceState)
    // save State in a temporary
    s := self.State
    // top-most initial transition
    Trigger(hsm, self.SourceState, event)
    // initial transition must go *one* level deep
    AssertEqual(s, Trigger(hsm, self.State, StdEvents[EventEmpty]))
    // update the termporary
    s = self.State
    // enter the state
    Trigger(hsm, s, StdEvents[EventEntry])
    for Trigger(hsm, s, StdEvents[EventInit]) == nil { // init handled?
        // initial transition must go *one* level deep
        AssertEqual(s, Trigger(hsm, self.State, StdEvents[EventEmpty]))
        s = self.State
        // enter the substate
        Trigger(hsm, s, StdEvents[EventEntry])
    }
    // we are in well-initialized state now
}

// Dispatch() is part of interface HSM.
func (self *StdHSM) Dispatch(event Event) {
    self.Dispatch2(self, event)
}

// Dispatch2() is a helper function to dispatch event to the concrete HSM.
func (self *StdHSM) Dispatch2(hsm HSM, event Event) {
    // Use `SourceState' to record the state which handle the event indeed(which
    // could be super, super-super, ... state).
    // `State' would stay unchange pointing at the current(most concrete) state.
    for self.SourceState = self.State; self.SourceState != nil; {
        self.SourceState = Trigger(hsm, self.SourceState, event)
    }
}

// GetState() is part of interface HSM.
func (self *StdHSM) GetState() State {
    return self.State
}

// IsIn() is part of interface HSM.
func (self *StdHSM) IsIn(stateID string) bool {
    state := self.StateTable[stateID]
    return self.isIn(state)
}

// isIn() is a helper function for IsIn().
// It will traverse from current state up to top state to find
// the specified state, util it finds a match or reachs top with failture.
func (self *StdHSM) isIn(state State) bool {
    // nagivate from current state up to all super state and
    // try to find specified `state'
    s := self.State
    for ; s != nil; s = Trigger(self, self.State, StdEvents[EventEmpty]) {
        if s == state {
            // a match is found
            return true
        }
    }
    // no match found
    return false
}

// QInit() is part of interface HSM.
func (self *StdHSM) QInit(targetStateID string) {
    target := self.LookupState(targetStateID)
    self.qinit(target)
}

// qinit() is a helper function for QInit().
func (self *StdHSM) qinit(state State) {
    self.State = state
}

// QTran() is part of interface HSM.
func (self *StdHSM) QTran(targetStateID string) {
    target := self.LookupState(targetStateID)
    self.QTranHSM(self, target)
}

// LookupState() search the specified state in state/name map.
func (self *StdHSM) LookupState(targetStateID string) State {
    AssertNotEqual(TopStateID, targetStateID)
    target, ok := self.StateTable[targetStateID]
    AssertTrue(ok)
    return target
}

// QTran2() is a helper function for QTran().
// It's separated from QTran() in order to deliver the concrete HSM
// (which is the first arguemnt of QTran2()) rather than just
// the embedded StdHSM to the state transfer procedure.
func (self *StdHSM) QTranHSM(hsm HSM, target State) {
    self.QTranHSMOnEvents(
        hsm,
        target,
        StdEvents[EventEntry],
        StdEvents[EventInit],
        StdEvents[EventExit])
}

// QTranOnEvent() is a variant function of QTran().
// Instead of dispatching the default events of
// `EventEntry'/`EventInit'/`EventExit', this function would dispatch
// the given event along the state transfer procedure.
func (self *StdHSM) QTranOnEvent(targetStateID string, event Event) {
    target := self.LookupState(targetStateID)
    self.QTranHSMOnEvent(self, target, event)
}

func (self *StdHSM) QTranHSMOnEvent(hsm HSM, target State, event Event) {
    self.QTranHSMOnEvents(hsm, target, event, event, event)
}

// QTranOnEvents() is the implementation of QTran* functions.
func (self *StdHSM) QTranHSMOnEvents(
    hsm HSM, target State, entryEvent, initEvent, exitEvent Event) {

    var p, q, s State
    for s := self.State; s != self.SourceState; {
        // we are about to dereference `s'
        AssertNotEqual(nil, s)
        t := Trigger(hsm, s, exitEvent)
        if t != nil {
            s = t
        } else {
            s = Trigger(hsm, s, StdEvents[EventEmpty])
        }
    }

    stateChain := list.New()
    stateChain.Init()

    stateChain.PushBack(nil)
    stateChain.PushBack(target)

    // (a) check `SourceState' == `target' (transition to self)
    if self.SourceState == target {
        Trigger(hsm, self.SourceState, exitEvent)
        goto inLCA
    }
    // (b) check `SourceState' == `target.Super()'
    p = Trigger(hsm, target, StdEvents[EventEmpty])
    if self.SourceState == p {
        goto inLCA
    }
    // (c) check `SourceState.Super()' == `target.Super()' (most common)
    q = Trigger(hsm, self.SourceState, StdEvents[EventEmpty])
    if q == p {
        Trigger(hsm, self.SourceState, exitEvent)
        goto inLCA
    }
    // (d) check `SourceState.Super()' == `target'
    if q == target {
        Trigger(hsm, self.SourceState, exitEvent)
        stateChain.Remove(stateChain.Back())
        goto inLCA
    }
    // (e) check rest of `SourceState' == `target.Super().Super()...'  hierarchy
    stateChain.PushBack(p)
    s = Trigger(hsm, p, StdEvents[EventEmpty])
    for s != nil {
        if self.SourceState == s {
            goto inLCA
        }
        stateChain.PushBack(s)
        s = Trigger(hsm, s, StdEvents[EventEmpty])
    }
    // exit source state
    Trigger(hsm, self.SourceState, exitEvent)
    // (f) check rest of `SourceState.Super()' == `target.Super().Super()...'
    for lca := stateChain.Back(); lca != nil; lca = lca.Prev() {
        if q == lca.Value {
            // do not enter the LCA
            stateChain = ListTruncate(stateChain, lca)
            goto inLCA
        }
    }
    // (g) check each `SourceState.Super().Super()...' for target...
    for s = q; s != nil; s = Trigger(hsm, s, StdEvents[EventEmpty]) {
        for lca := stateChain.Back(); lca != nil; lca = lca.Prev() {
            if s == lca.Value {
                stateChain = ListTruncate(stateChain, lca)
                goto inLCA
            }
        }
        Trigger(hsm, s, exitEvent)
    }
    // malformed HSM
    AssertTrue(false)
inLCA: // now we are in the LCA of `SourceState' and `target'
    // retrace the entry path in reverse order
    for e := stateChain.Back(); e != nil && e.Value != nil; {
        s, ok := e.Value.(State)
        AssertEqual(true, ok)
        Trigger(hsm, s, entryEvent) // enter `s' state
        stateChain.Remove(stateChain.Back())
        e = stateChain.Back()
    }
    // update current state
    self.State = target
    for Trigger(hsm, target, initEvent) == nil {
        // initial transition must go *one* level deep
        AssertEqual(target, Trigger(hsm, self.State, StdEvents[EventEmpty]))
        target = self.State
        Trigger(hsm, target, entryEvent)
    }
}
