// Improved EventManager with better target discrimination
class EventManager {
    constructor() {
        this.handlers = {
            afterSwap: new Map(),
            beforeSwap: new Map(),
            beforeRequest: new Map(),
            afterRequest: new Map(),
            configRequest: new Map(),
            responseError: new Map(),
        };
        this.initialized = false;
        this.initializedHandlers = {
            afterSwap: false,
            beforeSwap: false,
            beforeRequest: false,
            afterRequest: false,
            configRequest: false,
            responseError: false,
        };

        this.initializeListeners();
    }

    // Enhanced register method with more options
    register(eventType, targetSelector, handler, opts = {}) {
        if (!this.handlers[eventType]) {
            this.handlers[eventType] = new Map();
            if (!this.initializedHandlers[eventType]) {
                this.setupListener(eventType);
            }
        }

        const handlerConfig = {
            handler,
            once: opts.once || false,
            direct: opts.direct || false, // Only trigger if the target directly matches (includes internal children)
            preventBubble: opts.preventBubble || false, // Prevent handling bubbled events from external components
            preventExternal: opts.preventExternal || false, // Prevent external components from triggering
            includeChildren: opts.includeChildren !== false, // Include child elements (default true)
            excludeSelectors: opts.excludeSelectors || [], // Exclude certain child selectors
            triggerSource: opts.triggerSource || null, // Only trigger from specific sources
            useMatchingName: opts.useMatchingName || false, // Use matching name instead of selector
        };

        const currentHandlers =
            this.handlers[eventType].get(targetSelector) ?? [];
        this.handlers[eventType].set(targetSelector, [
            ...currentHandlers,
            handlerConfig,
        ]);

        return this; // Allow chaining
    }

    // Initialize event listeners
    initializeListeners() {
        // HTMX Events
        const htmxEvents = [
            "htmx:afterSwap",
            "htmx:beforeSwap",
            "htmx:beforeRequest",
            "htmx:afterRequest",
            "htmx:configRequest",
            "htmx:responseError",
        ];

        htmxEvents.forEach(this.setupListener.bind(this));

        // Regular DOM events
        document.addEventListener("click", (e) => {
            this.handleClicks(e);
        });

        this.initialized = true;
    }

    setupListener(eventName) {
        const eventType = eventName.replace("htmx:", "");
        document.body.addEventListener(eventName, (e) => {
            this.routeEvent(eventType, e);
        });
        this.initializedHandlers[eventType] = true;
    }

    routeEvent(eventType, event) {
        const handlerMap = this.handlers[eventType];
        if (!handlerMap) return;

        // Get all relevant targets from the event
        const eventTargets = this.extractEventTargets(event);

        // Track which handlers have been executed to prevent duplicates
        const executedHandlers = new Set();

        for (const [selector, handlerConfigs] of handlerMap) {
            // Check if this selector matches any of our targets
            const matchResult = this.findMatchingTarget(
                eventTargets,
                selector,
                handlerConfigs,
            );

            if (matchResult.matches) {
                // Process handlers for this selector
                const updatedHandlers = this.processHandlers(
                    handlerConfigs,
                    event,
                    matchResult,
                    eventType,
                    selector,
                    executedHandlers,
                );

                if (updatedHandlers !== null) {
                    handlerMap.set(selector, updatedHandlers);
                }
            }
        }
    }

    // Extract all possible targets from an event
    extractEventTargets(event) {
        const targets = {
            primary: null,
            trigger: null,
            swap: null,
            allTargets: [],
        };

        // Primary target (what actually got swapped/targeted)
        targets.primary = event.detail?.target || event.target;

        // Trigger element (what initiated the request)
        targets.trigger = event.detail?.elt || event.detail?.requestConfig?.elt;

        // Swap target (for afterSwap events)
        targets.swap = event.detail?.target;

        // Collect all unique targets
        const allTargets = [
            targets.primary,
            targets.trigger,
            targets.swap,
        ].filter(Boolean);
        targets.allTargets = [...new Set(allTargets)];

        return targets;
    }

    // Find if any target matches the selector
    findMatchingTarget(eventTargets, selector, event) {
        const result = {
            matches: false,
            target: null,
            isDirect: false,
            isChild: false,
            isInternalChild: false, // Child element within the target (like SVG in button)
            isExternalTrigger: false, // Different element triggering parent (like add-to-cart in grid)
            triggerElement: null,
        };

        // Check each target
        for (const target of eventTargets.allTargets) {
            if (!target) continue;

            // Direct match
            if (this.elementMatchesSelector(target, selector)) {
                result.matches = true;
                result.target = target;
                result.triggerElement = eventTargets.trigger;

                // Determine if this is truly direct or triggered by an internal child
                if (eventTargets.trigger === target) {
                    result.isDirect = true;
                } else if (
                    eventTargets.trigger &&
                    target.contains(eventTargets.trigger)
                ) {
                    // The trigger is a child of the matched target
                    result.isDirect = true; // Treat as direct since it's internal
                    result.isInternalChild = true;
                }
                break;
            }

            // Check if a child element triggered this (external trigger)
            if (eventTargets.trigger && eventTargets.trigger !== target) {
                const parent = this.findParentMatching(
                    eventTargets.trigger,
                    selector,
                );
                if (parent === target) {
                    result.matches = true;
                    result.target = target;
                    result.isChild = true;
                    result.triggerElement = eventTargets.trigger;

                    // Check if this is an external trigger (different component)
                    // by looking for data attributes or specific selectors
                    if (
                        this.isExternalComponent(eventTargets.trigger, target)
                    ) {
                        result.isExternalTrigger = true;
                    }
                    break;
                }
            }
        }

        return result;
    }

    // Determine if trigger element is an external component
    isExternalComponent(trigger, parent) {
        // Check if trigger has its own HTMX attributes (indicating it's a separate component)
        if (
            trigger.hasAttribute("hx-get") ||
            trigger.hasAttribute("hx-post") ||
            trigger.hasAttribute("hx-put") ||
            trigger.hasAttribute("hx-delete") ||
            trigger.hasAttribute("hx-patch")
        ) {
            return true;
        }

        // Check for specific data attributes that indicate separate components
        const componentAttributes = [
            "data-add-to-cart",
            "data-quick-view",
            "data-remove-item",
            "data-modal-trigger",
            "data-dropdown-trigger",
        ];

        for (const attr of componentAttributes) {
            if (trigger.hasAttribute(attr)) {
                return true;
            }
        }

        // Check if trigger is a form or interactive element with its own functionality
        if (
            trigger.matches(
                'form, [role="button"][hx-target], .modal, .dropdown',
            )
        ) {
            return true;
        }

        return false;
    }

    // Check if element matches selector
    elementMatchesSelector(element, selector) {
        if (!element) return false;

        // Handle special selectors
        switch (selector) {
            case "wizard-modal":
                return element.id === "wizard-modal";
            case "product-modal":
                return element.id === "product-modal";
            case "cart-sidebar":
                return element.id === "cart-sidebar";
            case "products-grid":
                return (
                    element.id === "products" || element.closest("#products")
                );
            case "category-filter":
                return element.matches?.("[data-category-filter]");
            case "add-to-cart":
                return element.matches?.("[data-add-to-cart]");
            case "wizard-step":
                return window.location.pathname?.includes("/wizard/step/");
            case "search-input":
                return element.matches?.("[data-search-input]");
            default:
                // Use CSS selector
                try {
                    return element.matches?.(selector);
                } catch {
                    return false;
                }
        }
    }

    // Find parent element matching selector
    findParentMatching(element, selector) {
        if (!element) return null;

        try {
            // Map special selectors to CSS selectors
            const cssSelector = this.mapToCssSelector(selector);
            return element.closest(cssSelector);
        } catch {
            return null;
        }
    }

    // Map special selectors to CSS selectors
    mapToCssSelector(selector) {
        const mapping = {
            "wizard-modal": "#wizard-modal",
            "product-modal": "#product-modal",
            "cart-sidebar": "#cart-sidebar",
            "products-grid": "#products",
            "category-filter": "[data-category-filter]",
            "add-to-cart": "[data-add-to-cart]",
            "search-input": "[data-search-input]",
        };

        return mapping[selector] || selector;
    }

    // Process handlers with enhanced discrimination
    processHandlers(
        handlerConfigs,
        event,
        matchResult,
        eventType,
        selector,
        executedHandlers,
    ) {
        let updatedHandlers = null;
        const configsToKeep = [];

        for (const config of handlerConfigs) {
            // Create unique handler ID to prevent duplicate execution
            const handlerId = `${selector}-${handlerConfigs.indexOf(config)}`;

            if (executedHandlers.has(handlerId)) {
                configsToKeep.push(config);
                continue;
            }

            let shouldExecute = true;

            // Check if we should execute based on config options
            if (
                config.direct &&
                !matchResult.isDirect &&
                !matchResult.isInternalChild
            ) {
                // Don't execute if not direct AND not an internal child (like SVG in button)
                shouldExecute = false;
            }

            if (
                config.preventBubble &&
                matchResult.isChild &&
                !matchResult.isInternalChild
            ) {
                // Only prevent bubble from external triggers, not internal children
                shouldExecute = false;
            }

            // New option: preventExternal - prevents external components from triggering
            if (config.preventExternal && matchResult.isExternalTrigger) {
                shouldExecute = false;
            }

            // Check if trigger element is excluded
            if (
                config.excludeSelectors.length > 0 &&
                matchResult.triggerElement
            ) {
                for (const excludeSelector of config.excludeSelectors) {
                    if (
                        matchResult.triggerElement.matches?.(excludeSelector) ||
                        matchResult.triggerElement.closest?.(excludeSelector)
                    ) {
                        shouldExecute = false;
                        break;
                    }
                }
            }

            // Check trigger source if specified
            if (config.triggerSource && matchResult.triggerElement) {
                if (
                    !matchResult.triggerElement.matches?.(
                        config.triggerSource,
                    ) &&
                    !matchResult.triggerElement.closest?.(config.triggerSource)
                ) {
                    shouldExecute = false;
                }
            }

            if (shouldExecute) {
                try {
                    // Execute handler with enhanced context
                    config.handler(event, {
                        isDirect: matchResult.isDirect,
                        isChild: matchResult.isChild,
                        isInternalChild: matchResult.isInternalChild,
                        isExternalTrigger: matchResult.isExternalTrigger,
                        triggerElement: matchResult.triggerElement,
                        targetElement: matchResult.target,
                    });

                    executedHandlers.add(handlerId);
                } catch (error) {
                    console.error(
                        `Error in ${eventType} handler for ${selector}:`,
                        error,
                    );
                }
            }

            // Keep handler if not a one-time handler
            if (!config.once || !shouldExecute) {
                configsToKeep.push(config);
            } else {
                updatedHandlers = configsToKeep;
            }
        }

        return updatedHandlers || configsToKeep;
    }

    // Handle click events
    handleClicks(event) {
        const target = event.target.matches("[data-click-handler-selector]")
            ? event.target
            : event.target.closest("[data-click-handler-selector]");

        if (!target) return;

        const handlerSelector = target.dataset.clickHandlerSelector;
        const selectors = handlerSelector.split(",");

        for (const selector of selectors) {
            this.triggerHandler("click", selector, event);
        }
    }

    triggerHandler(eventType, selector, event) {
        const key = `click-${selector}`;
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

    // Utility method to debug event flow
    debug(enabled = true) {
        if (enabled) {
            this._originalRouteEvent = this.routeEvent;
            this.routeEvent = function (eventType, event) {
                console.group(`Event: ${eventType}`);
                console.log("Event details:", {
                    target: event.detail?.target,
                    trigger: event.detail?.elt,
                    path: event.detail?.path,
                    requestConfig: event.detail?.requestConfig,
                });
                console.groupEnd();
                return this._originalRouteEvent.call(this, eventType, event);
            };
        } else if (this._originalRouteEvent) {
            this.routeEvent = this._originalRouteEvent;
        }
    }
}

// Initialize
window.eventManager = new EventManager();

// Example usage:
/*
// EXAMPLE 1: Button with SVG icon - handler WILL fire when clicking the SVG
eventManager.register('afterRequest', '[data-add-to-cart]', (event, context) => {
    console.log('Add to cart clicked');
    // This WILL fire even when clicking the SVG icon inside the button
}, {
    direct: true  // Still works with internal children like SVG
});

// EXAMPLE 2: Products grid - WON'T fire when add-to-cart is clicked
eventManager.register('afterSwap', 'products-grid', (event, context) => {
    console.log('Products grid updated');
    if (!context.isExternalTrigger) {
        animateProductCards();  // Only animate for direct grid updates
    }
}, {
    preventExternal: true,  // Prevent external components from triggering
    excludeSelectors: ['[data-add-to-cart]']  // Extra safety
});

// EXAMPLE 3: More nuanced control
eventManager.register('afterSwap', '#my-container', (event, context) => {
    // Context provides detailed information:
    // - context.isDirect: true if element was the direct target
    // - context.isInternalChild: true if triggered by child like SVG in button
    // - context.isExternalTrigger: true if triggered by separate component
    
    if (context.isInternalChild) {
        console.log('Triggered by internal child element (like SVG in button)');
    } else if (context.isExternalTrigger) {
        console.log('Triggered by external component (like add-to-cart in grid)');
    } else if (context.isDirect) {
        console.log('Direct target match');
    }
});

// EXAMPLE 4: Click handler that works with child elements
eventManager.registerClick('my-button', (event) => {
    // This will fire when clicking the button OR any of its children (text, icons, etc.)
    console.log('Button clicked');
});
*/
