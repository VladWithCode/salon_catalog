(function() {
    if (!window.gsap) {
        console.error('PopupManager requires GSAP to be loaded first.');
        return;
    }
    // Popup Manager Class for reusable popup functionality
    class PopupManager {
        constructor(config) {
            this.popup = document.getElementById(config.popupId);
            if (!this.popup) {
                console.error('PopupManager: Popup element not found.');
                return;
            }
            this.openBtns = document.querySelectorAll(config.openBtnsSelector);
            this.openBtn = document.getElementById(config.openBtnId);
            this.closeBtn = this.popup.querySelector(config.closeBtnSelector);
            this.contents = this.popup.querySelector(config.contentsSelector) || null;
            this.container = this.popup.querySelector(config.containerSelector);

            this.previouslyFocusedElement = null;
            this.isAnimating = false;
            this.openTimeline = gsap.timeline({ paused: true });
            this.closeTimeline = gsap.timeline({ paused: true });

            if (typeof config.handleOpenClick === 'function') {
                this.handleOpenClick = config.handleOpenClick.bind(this);
            }
            if (typeof config.handleCloseClick === 'function') {
                this.handleCloseClick = config.handleCloseClick.bind(this);
            }

            this.init();
        }

        init() {
            this.initAnimations();
            this.initHoverAnimations();
            this.bindEvents();
        }

        initAnimations() {
            // Set initial states
            gsap.set(this.popup, {
                visibility: 'hidden',
                opacity: 0,
            });

            gsap.set(this.container, {
                scale: 0,
            });

            gsap.set(this.contents, {
                scale: 0.8,
                opacity: 0,
                rotation: -5,
            });

            gsap.set(this.closeBtn, {
                scale: 0,
                opacity: 0,
                rotation: 180,
            });

            // Open animation timeline
            this.openTimeline
                .set(this.popup, { visibility: 'visible' })
                .to(this.popup, {
                    opacity: 1,
                    duration: 0.3,
                    ease: "power2.out"
                })
                .to(this.container, {
                    scale: 1,
                    duration: 0.6,
                    ease: "back.out(1.7)"
                }, "-=0.1")
                .to(this.contents, {
                    scale: 1,
                    opacity: 1,
                    rotation: 0,
                    duration: 0.6,
                    ease: "back.out(1.7)"
                }, "-=0.25")
                .to(this.closeBtn, {
                    scale: 1,
                    opacity: 1,
                    rotation: 0,
                    duration: 0.4,
                    ease: "back.out(1.7)"
                }, "-=0.3");

            // Close animation timeline
            this.closeTimeline
                .to(this.closeBtn, {
                    scale: 0,
                    opacity: 0,
                    rotation: -180,
                    duration: 0.2,
                    ease: "power2.in"
                })
                .to(this.contents, {
                    scale: 0.8,
                    opacity: 0,
                    rotation: 5,
                    duration: 0.4,
                    ease: "back.in(1.7)"
                }, "-=0.1")
                .to(this.container, {
                    scale: 0,
                    duration: 0.6,
                    ease: "back.out(1.7)"
                }, "-=0.2")
                .to(this.popup, {
                    opacity: 0,
                    duration: 0.3,
                    ease: "power2.in"
                }, "-=0.2")
                .set(this.popup, { visibility: 'hidden' });
        }

        initHoverAnimations() {
            // Open button hover effect
            //if (this.openBtn) {
            //    this.openBtn.addEventListener('mouseenter', () => {
            //        gsap.to(this.openBtn, {
            //            scale: 1.02,
            //            duration: 0.2,
            //            ease: "power2.out"
            //        });
            //    });
            //
            //    this.openBtn.addEventListener('mouseleave', () => {
            //        gsap.to(this.openBtn, {
            //            scale: 1,
            //            duration: 0.2,
            //            ease: "power2.out"
            //        });
            //    });
            //}

            // Close button hover effect
            this.closeBtn.addEventListener('mouseenter', () => {
                gsap.to(this.closeBtn, {
                    scale: 1.1,
                    rotation: 90,
                    duration: 0.3,
                    ease: "back.out(1.7)"
                });
            });

            this.closeBtn.addEventListener('mouseleave', () => {
                gsap.to(this.closeBtn, {
                    scale: 1,
                    rotation: 0,
                    duration: 0.3,
                    ease: "back.out(1.7)"
                });
            });
        }

        bindEvents() {
            if (this.openBtns) {
                for (const btn of this.openBtns) {
                    btn.addEventListener('click', e => {
                        this.open()

                        if (typeof this.handleOpenClick === 'function') {
                            this.handleOpenClick(e);
                        }
                    });
                }
            }
            if (this.openBtn) {
                this.openBtn.addEventListener('click', e => {
                    this.open()

                    if (typeof this.handleOpenClick === 'function') {
                        this.handleOpenClick(e);
                    }
                });
            }

            this.closeBtn.addEventListener('click', e => {
                this.close()

                if (typeof this.handleCloseClick === 'function') {
                    this.handleCloseClick(e);
                }
            });
        }

        open() {
            if (this.isAnimating) return;

            this.isAnimating = true;
            this.previouslyFocusedElement = document.activeElement;
            document.body.style.overflow = 'hidden';

            this.openTimeline.restart();

            setTimeout(() => {
                this.closeBtn.focus();
                document.addEventListener('keydown', (e) => this.handleKeyDown(e));
                this.popup.addEventListener('click', (e) => this.handleBackdropClick(e));
                this.isAnimating = false;
            }, 100);
        }

        close() {
            if (this.isAnimating) return;

            this.isAnimating = true;
            document.removeEventListener('keydown', (e) => this.handleKeyDown(e));
            this.popup.removeEventListener('click', (e) => this.handleBackdropClick(e));

            this.closeTimeline.restart();

            this.closeTimeline.call(() => {
                document.body.style.overflow = '';
                if (this.previouslyFocusedElement) {
                    this.previouslyFocusedElement.focus();
                }
                this.isAnimating = false;
            });
        }

        handleKeyDown(event) {
            if (event.key === 'Escape') {
                this.close();
            }

            if (event.key === 'Tab') {
                const focusableElements = this.popup.querySelectorAll(
                    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
                );
                const firstElement = focusableElements[0];
                const lastElement = focusableElements[focusableElements.length - 1];

                if (event.shiftKey) {
                    if (document.activeElement === firstElement) {
                        event.preventDefault();
                        lastElement.focus();
                    }
                } else {
                    if (document.activeElement === lastElement) {
                        event.preventDefault();
                        firstElement.focus();
                    }
                }
            }
        }

        handleBackdropClick(event) {
            if (event.target === this.popup) {
                this.close();
            }
        }
    }

    window.PopupManager = PopupManager;
})()
