package tsdb

// AddFilter adds new filter entry
func (pr *Process) AddFilter(filter Filter) IDt {
	defer func() {
		pr.apiMutex.Unlock()
	}()
	pr.apiMutex.Lock()
	filter.ID = GetNewID(pr.Config.Filters)
	pr.Config.Filters = append(pr.Config.Filters, filter)
	return filter.ID
}

// RemoveFilter removes 1 filter entry by ID
func (pr *Process) RemoveFilter(ID IDt) {
	defer func() {
		pr.apiMutex.Unlock()
	}()
	pr.apiMutex.Lock()
	for i := range pr.Config.Filters {
		if pr.Config.Filters[i].ID == ID {
			pr.Config.Filters = append(pr.Config.Filters[:i], pr.Config.Filters[i+1:]...)
		}
	}
}

// AddSelector adds new selector
func (pr *Process) AddSelector(selector Selector) IDt {
	defer func() {
		pr.apiMutex.Unlock()
	}()
	pr.apiMutex.Lock()
	selector.ID = GetNewID(pr.Config.Filters)
	pr.Config.Selectors = append(pr.Config.Selectors, selector)
	pr.mqttTransport.Subscribe(selector.Topic)
	return selector.ID
}

// RemoveSelector removes 1 selector entry
func (pr *Process) RemoveSelector(ID IDt) {
	defer func() {
		pr.apiMutex.Unlock()
	}()
	pr.apiMutex.Lock()
	for i := range pr.Config.Selectors {
		if pr.Config.Selectors[i].ID == ID {
			pr.mqttTransport.Unsubscribe(pr.Config.Selectors[i].Topic)
			pr.Config.Selectors = append(pr.Config.Selectors[:i], pr.Config.Selectors[i+1:]...)
		}
	}
}


// GetFilters returns all filters
func (pr *Process) GetFilters() []Filter {
	return pr.Config.Filters
}

// GetSelectors return all selectors
func (pr *Process) GetSelectors() []Selector {
	return pr.Config.Selectors
}
