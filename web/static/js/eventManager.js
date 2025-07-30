// Create a new shared script component
class EventManager {
    constructor() {
        this.handlers = {
            afterSwap: new Map(),
            beforeSwap: new Map(),
            beforeRequest: new Map(),
            afterRequest: new Map(),
            configRequest: new Map(),
            responseError: new Map()
        };

        this.initializeListeners();
    }

    // Register handlers for different target types
    register(eventType, targetSelector, handler) {
        if (!this.handlers[eventType]) {
            this.handlers[eventType] = new Map();
        }
        const currentHandlers = this.handlers[eventType].get(targetSelector) ?? [];

        this.handlers[eventType].set(targetSelector, [...currentHandlers, handler]);
        return this; // Allow chaining
    }

    // Single event listeners that route to appropriate handlers
    initializeListeners() {
        // HTMX After Swap Handler
        document.body.addEventListener('htmx:afterSwap', (e) => {
            this.routeEvent('afterSwap', e);
        });

        document.body.addEventListener('htmx:beforeSwap', (e) => {
            this.routeEvent('beforeSwap', e);
        });

        // HTMX Before Request Handler  
        document.body.addEventListener('htmx:beforeRequest', (e) => {
            this.routeEvent('beforeRequest', e);
        });

        // HTMX After Request Handler
        document.body.addEventListener('htmx:afterRequest', (e) => {
            this.routeEvent('afterRequest', e);
        });

        // HTMX Config Request Handler
        document.body.addEventListener('htmx:configRequest', (e) => {
            this.routeEvent('configRequest', e);
        });

        // HTMX Response Error Handler
        document.body.addEventListener('htmx:responseError', (e) => {
            this.routeEvent('responseError', e);
        });

        // Regular DOM events can also be centralized
        document.addEventListener('click', (e) => {
            this.handleClicks(e);
        });
    }

    routeEvent(eventType, event) {
        const handlerMap = this.handlers[eventType];
        if (!handlerMap) return;

        // Try to match handlers by target ID, class, or selector
        for (const [selector, handlers] of handlerMap) {
            if (this.matchesSelector(event, selector)) {
                try {
                    for (const handler of handlers) {
                        handler(event);
                    }
                } catch (error) {
                    console.error(`Error in ${eventType} handler for ${selector}:`, error);
                }
            }
        }
    }

    matchesSelector(event, selector) {
        const target = event.detail?.target 
            || event.detail?.elt 
            || event.detail?.requestConfig?.elt
            || event.target;

        if (!target) {
            return false;
        }

        switch (selector) {
            case 'wizard-modal':
                return target.id === 'wizard-modal';

            case 'product-modal':
                return target.id === 'product-modal';

            case 'cart-sidebar':
                return target.id === 'cart-sidebar';

            case 'products-grid':
                return target.id === 'products' || event.detail.requestConfig.elt.closest('#products');

            case 'category-filter':
                return event.detail?.elt?.matches?.('[data-category-filter]') ||
                    event.target.matches?.('[data-category-filter]');

            case 'add-to-cart':
                return event.detail?.elt?.matches?.('[data-add-to-cart]') ||
                    event.target.matches?.('[data-add-to-cart]');

            case 'wizard-step':
                return event.detail?.path?.includes('/wizard/step/');

            case 'search-input':
                return event.detail?.elt?.matches?.('[data-search-input]');

            default:
                // Fallback to CSS selector matching
                return target.matches(selector)
                    || Boolean(event.detail.elt?.closest(selector))
                    || event.detail.requestConfig?.elt?.closest(selector);
        }
    }

    handleClicks(event) {
        const target = event.target.matches('[data-click-handler-selector]') 
            ? event.target 
            : event.target.closest('[data-click-handler-selector]');

        if (!target) {
            return;
        }

        const handlerSelector = target.dataset.clickHandlerSelector;
        const selectors = handlerSelector.split(',');
        for (const selector of selectors) {
            this.triggerHandler('click', selector, event);
        }
    }

    triggerHandler(eventType, selector, event) {
        const key = `${eventType}-${selector}`;
        const handler = this.handlers[key];
        if (handler) {
            handler(event);
        }
    }

    // Utility method to register click handlers
    registerClick(selector, handler) {
        const key = `click-${selector}`;
        this.handlers[key] = handler;
        return this;
    }
}

window.eventManager = new EventManager();
