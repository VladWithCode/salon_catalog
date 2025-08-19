// Wizard Modal Animation Manager
class WizardModalManager {
    constructor() {
        this.modalContainer = null;
        this.modalElement = null;
        this.backdrop = null;
        this.isAnimating = false;

        this.init();
    }

    init() {
        // Wait for DOM to be ready
        if (document.readyState === "loading") {
            document.addEventListener("DOMContentLoaded", () => this.setup());
        } else {
            this.setup();
        }
    }

    setup() {
        this.modalContainer = document.getElementById(
            "wizards-modal-container",
        );
        this.modalElement = document.getElementById("wizards-modal");
        this.backdrop = document.getElementById("wizards-modal-backdrop");

        if (!this.modalContainer || !this.modalElement) {
            console.warn(
                "Wizard modal elements not found, will try again after content loads",
            );
            return;
        }

        this.registerEventHandlers();
        this.registerClickHandlers();
    }

    registerEventHandlers() {
        // Register afterSwap event for when modal content is loaded
        window.eventManager.register(
            "afterSwap",
            "wizards-modal",
            (event, context) => {
                console.log("steps modal swap");
                this.handleModalContentSwap(event, context);
            },
            {
                direct: true,
                preventExternal: true,
            },
        );

        // Also listen for direct wizard modal container updates
        window.eventManager.register(
            "afterSwap",
            "wizards-modal-container",
            (event, context) => {
                console.log("steps modal swap");
                this.handleModalContainerSwap(event, context);
            },
            {
                direct: true,
                preventExternal: true,
            },
        );

        // Listen for wizard steps modal content updates
        window.eventManager.register(
            "afterSwap",
            "#wizard-steps-modal",
            (event, context) => {
                console.log("steps modal swap");
                this.handleStepsModalSwap(event, context);
            },
            {
                direct: true,
                preventExternal: true,
            },
        );
    }

    registerClickHandlers() {
        // Handle modal toggle buttons (open/close)
        window.eventManager.registerClick(
            "[data-wizards-modal-toggle]",
            (event) => {
                const action = event.target.closest(
                    "[data-wizards-modal-toggle]",
                ).dataset.wizardsModalToggle;

                if (action === "open") {
                    this.openModal();
                } else if (action === "close") {
                    this.closeModal();
                }
            },
        );

        // Handle steps modal close function (referenced in templates)
        window.closeStepsModal = () => {
            this.closeStepsModal();
        };

        // Handle opening steps modal when "Add Step" button is clicked
        window.eventManager.registerClick(
            '[hx-get*="/pasos/modal/"]',
            (event) => {
                // This will be handled by the afterSwap event, but we can add any pre-opening logic here
            },
        );
    }

    handleModalContentSwap(event, context) {
        // Re-setup modal elements in case they changed
        this.modalContainer = document.getElementById(
            "wizards-modal-container",
        );
        this.modalElement = document.getElementById("wizards-modal");
        this.backdrop = document.getElementById("wizards-modal-backdrop");

        if (!this.modalContainer || !this.modalElement) {
            console.warn("Modal elements not found after content swap");
            return;
        }

        // If modal content was swapped and modal is supposed to be open, animate it in
        if (
            context.triggerElement &&
            context.triggerElement.matches(
                '[hx-get*="/panel/asistentes/modal/"]',
            )
        ) {
            this.animateModalIn();
        }
    }

    handleModalContainerSwap(event, context) {
        // Handle when the entire modal container is updated
        this.setup(); // Re-setup everything
    }

    handleStepsModalSwap(event, context) {
        // Handle when wizard steps modal content is loaded
        const stepsModal = document.getElementById("wizard-steps-modal");
        if (!stepsModal) return;

        // Animate the steps modal in
        gsap.set(stepsModal, {
            display: "flex",
            opacity: 0,
            pointerEvents: "none",
        });

        const modalContent = stepsModal.querySelector(".relative");
        if (modalContent) {
            gsap.set(modalContent, {
                scale: 0.9,
                y: 20,
                opacity: 0,
            });

            const tl = gsap.timeline({
                onComplete: () => {
                    stepsModal.style.pointerEvents = "auto";
                },
            });

            tl.to(stepsModal, {
                opacity: 1,
                duration: 0.15,
                ease: "power2.out",
            }).to(
                modalContent,
                {
                    scale: 1,
                    y: 0,
                    opacity: 1,
                    duration: 0.25,
                    ease: "back.out(1.1)",
                },
                "-=0.1",
            );
        }
    }

    openModal() {
        if (this.isAnimating || !this.modalContainer || !this.modalElement) {
            return;
        }

        this.isAnimating = true;

        // Set initial state for animation
        gsap.set(this.modalContainer, {
            display: "flex",
            opacity: 0,
            pointerEvents: "none",
        });

        gsap.set(this.modalElement, {
            opacity: 0,
            y: 48, // translate-y-12 equivalent
            scale: 0.95,
        });

        // Animate in
        const tl = gsap.timeline({
            onComplete: () => {
                this.isAnimating = false;
                this.modalContainer.style.pointerEvents = "auto";
            },
        });

        tl.to(this.modalContainer, {
            opacity: 1,
            duration: 0.15,
            ease: "power2.out",
        }).to(
            this.modalElement,
            {
                opacity: 1,
                y: 0,
                scale: 1,
                duration: 0.25,
                ease: "back.out(1.1)",
            },
            "-=0.1",
        );
    }

    animateModalIn() {
        if (this.isAnimating || !this.modalContainer || !this.modalElement) {
            return;
        }

        // This is called when content is swapped in, so we need to animate the entrance
        this.isAnimating = true;

        // Set container to visible immediately
        gsap.set(this.modalContainer, {
            display: "flex",
            opacity: 1,
            pointerEvents: "auto",
        });

        // Set initial state for modal content
        gsap.set(this.modalElement, {
            opacity: 0,
            y: 48,
            scale: 0.95,
        });

        // Animate modal content in
        gsap.to(this.modalElement, {
            opacity: 1,
            y: 0,
            scale: 1,
            duration: 0.3,
            ease: "back.out(1.2)",
            onComplete: () => {
                this.isAnimating = false;
            },
        });
    }

    closeModal() {
        if (this.isAnimating || !this.modalContainer || !this.modalElement) {
            return;
        }

        this.isAnimating = true;

        const tl = gsap.timeline({
            onComplete: () => {
                gsap.set(this.modalContainer, {
                    display: "none",
                    pointerEvents: "none",
                });
                this.isAnimating = false;

                // Clear modal content
                if (this.modalElement) {
                    this.modalElement.innerHTML = "";
                }
            },
        });

        tl.to(this.modalElement, {
            opacity: 0,
            y: 24,
            scale: 0.95,
            duration: 0.2,
            ease: "power2.in",
        }).to(
            this.modalContainer,
            {
                opacity: 0,
                duration: 0.15,
                ease: "power2.in",
            },
            "-=0.1",
        );
    }

    closeStepsModal() {
        // Handle the wizard steps modal (nested modal)
        const stepsModal = document.getElementById("wizard-steps-modal");
        if (!stepsModal) return;

        gsap.to(stepsModal, {
            opacity: 0,
            duration: 0.2,
            ease: "power2.in",
            onComplete: () => {
                gsap.set(stepsModal, {
                    display: "none",
                    pointerEvents: "none",
                });
                // Clear content
                const modalContent = stepsModal.querySelector(".relative");
                if (modalContent) {
                    modalContent.innerHTML = "";
                }
            },
        });
    }

    // Method to manually trigger modal open (useful for programmatic calls)
    show() {
        this.openModal();
    }

    // Method to manually trigger modal close (useful for programmatic calls)
    hide() {
        this.closeModal();
    }
}

// Initialize when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
    if (!window.wizardModalManager) {
        window.wizardModalManager = new WizardModalManager();
    }
});

// Also initialize immediately if DOM is already loaded
if (document.readyState !== "loading" && !window.wizardModalManager) {
    window.wizardModalManager = new WizardModalManager();
}

