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
        this.handlers[eventType].set(targetSelector, handler);
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
        const handlers = this.handlers[eventType];
        if (!handlers) return;

        // Try to match handlers by target ID, class, or selector
        for (const [selector, handler] of handlers) {
            if (this.matchesSelector(event, selector)) {
                try {
                    handler(event);
                } catch (error) {
                    console.error(`Error in ${eventType} handler for ${selector}:`, error);
                }
            }
        }
    }

    matchesSelector(event, selector) {
        const target = event.detail?.target || event.target;

        switch (selector) {
            case 'wizard-modal':
                return target?.id === 'wizard-modal';

            case 'product-modal':
                return target?.id === 'product-modal';

            case 'cart-sidebar':
                return target?.id === 'cart-sidebar';

            case 'products-grid':
                return target?.id === 'products' || event.detail.requestConfig.elt.closest('#products');

            case 'category-filter':
                return event.detail?.elt?.matches?.('[data-category-filter]') ||
                    event.target?.matches?.('[data-category-filter]');

            case 'add-to-cart':
                return event.detail?.elt?.matches?.('[data-add-to-cart]') ||
                    event.target?.matches?.('[data-add-to-cart]');

            case 'wizard-step':
                return event.detail?.path?.includes('/wizard/step/');

            case 'search-input':
                return event.detail?.elt?.matches?.('[data-search-input]');

            default:
                // Fallback to CSS selector matching
                return target?.matches?.(selector) || target?.closest?.(selector);
        }
    }

    handleClicks(event) {
        const target = event.target;

        // Route click events based on data attributes
        if (target.matches('[data-cart-toggle]') || target.closest('[data-cart-toggle]')) {
            this.triggerHandler('click', 'cart-toggle', event);
        }

        if (target.matches('[data-cart-close]') || target.closest('[data-cart-close]')) {
            this.triggerHandler('click', 'cart-close', event);
        }

        if (target.matches('[data-close-modal]') || target.closest('[data-close-modal]')) {
            this.triggerHandler('click', 'close-modal', event);
        }

        if (target.matches('[data-close-wizard]') || target.closest('[data-close-wizard]')) {
            this.triggerHandler('click', 'close-wizard', event);
        }

        if (target.matches('[data-browse-trigger]')) {
            this.triggerHandler('click', 'browse-trigger', event);
        }

        if (target.matches('#cart-backdrop')) {
            this.triggerHandler('click', 'cart-backdrop', event);
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
