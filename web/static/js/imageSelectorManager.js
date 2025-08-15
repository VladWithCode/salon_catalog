class ImageSelectorManager {
    constructor(config = {}) {
        this.config = {
            mode: config.mode || "single",
            title: config.title || "Seleccionar Imagen",
            maxSelection: config.maxSelection || 15,
            selectedIds: config.selectedIds || [],
            allowUpload: config.allowUpload !== false,
            updateEndpoint: config.updateEndpoint || null,
            ...config,
        };

        this.selectedImages = new Map();
        this.searchTimeout = null;
        this.modalElement = null;
        this.eventListeners = [];

        this.init();
    }

    init() {
        this.modalElement = document.getElementById("image-selector-modal");
        if (!this.modalElement) {
            console.error("ImageSelectorManager: Modal element not found");
            return;
        }

        this.loadPreselectedImages();
        this.setupEventListeners();
        this.updateSelectionUI();

        // Set initial tab state (select tab active by default)
        this.switchTab("select");
    }

    setupEventListeners() {
        if (!window.eventManager) {
            console.error("ImageSelectorManager: EventManager not available");
            return;
        }

        // Register HTMX event listeners using EventManager
        window.eventManager.register(
            "afterSwap",
            "#images-selector-grid",
            (event, context) => {
                this.handleGridUpdate(event, context);
            },
            {
                direct: true,
                preventExternal: true,
            },
        );

        // Register click handlers for modal elements
        this.addClickListener("[data-close-selector]", () => this.close());
        this.addClickListener("[data-confirm-selection]", () =>
            this.confirmSelection(),
        );
        this.addClickListener("[data-clear-selection]", () =>
            this.clearSelection(),
        );
        this.addClickListener("[data-switch-tab]", (event) => {
            const tab = event.target.getAttribute("data-tab");
            this.switchTab(tab);
        });
        this.addClickListener("[data-image-item]", (event) => {
            this.toggleImageSelection(
                event.target.closest("[data-image-item]"),
            );
        });
        this.addClickListener("[data-remove-from-selection]", (event) => {
            const imageId = event.target.getAttribute("data-image-id");
            this.removeFromSelection(imageId);
        });
        this.addClickListener("[data-load-page]", (event) => {
            const page = parseInt(event.target.getAttribute("data-page"));
            this.loadPage(page);
        });

        // Input event listeners
        this.addInputListener("[data-search-input]", (event) => {
            this.debounceSearch(event.target.value);
        });
        this.addChangeListener("[data-sort-select]", (event) => {
            this.updateSort(event.target.value);
        });

        // Modal backdrop click
        this.addClickListener("#image-selector-modal", (event) => {
            if (event.target === event.currentTarget) {
                this.close();
            }
        });
    }

    addClickListener(selector, handler) {
        const elements = this.modalElement.querySelectorAll(selector);
        elements.forEach((element) => {
            element.addEventListener("click", handler);
            this.eventListeners.push({ element, event: "click", handler });
        });
    }

    addInputListener(selector, handler) {
        const elements = this.modalElement.querySelectorAll(selector);
        elements.forEach((element) => {
            element.addEventListener("input", handler);
            this.eventListeners.push({ element, event: "input", handler });
        });
    }

    addChangeListener(selector, handler) {
        const elements = this.modalElement.querySelectorAll(selector);
        elements.forEach((element) => {
            element.addEventListener("change", handler);
            this.eventListeners.push({ element, event: "change", handler });
        });
    }

    loadPreselectedImages() {
        if (this.config.selectedIds && this.config.selectedIds.length > 0) {
            this.config.selectedIds.forEach((id) => {
                const element = this.modalElement.querySelector(
                    `[data-image-id="${id}"]`,
                );
                if (element) {
                    this.selectImageElement(element, id);
                }
            });
        }
    }

    handleGridUpdate(event, context) {
        // Re-apply selections after grid update
        this.selectedImages.forEach((imageData, imageId) => {
            const element = this.modalElement.querySelector(
                `[data-image-id="${imageId}"]`,
            );
            if (element) {
                this.applySelectionToElement(element, imageId);
            }
        });
        this.updateSelectionUI();
    }

    switchTab(tab) {
        // Get tab buttons
        const selectTab = this.modalElement.querySelector(
            '[data-switch-tab][data-tab="select"]',
        );
        const uploadTab = this.modalElement.querySelector(
            '[data-switch-tab][data-tab="upload"]',
        );

        // Get content containers
        const selectContent = this.modalElement.querySelector(
            "#images-selector-modal-select",
        );
        const uploadContent = this.modalElement.querySelector(
            "#images-selector-modal-upload-container",
        );

        if (tab === "select") {
            // Style select tab as active
            selectTab?.classList.remove(
                "text-gray-500",
                "border-transparent",
                "hover:text-gray-700",
                "hover:border-gray-300",
            );
            selectTab?.classList.add("text-gray-50", "bg-gray-700");

            // Style upload tab as inactive
            uploadTab?.classList.remove("text-gray-50", "bg-gray-700");
            uploadTab?.classList.add(
                "text-gray-500",
                "border-transparent",
                "hover:text-gray-700",
                "hover:border-gray-300",
            );

            // Show select content, hide upload content
            selectContent?.classList.remove("h-0", "overflow-hidden");
            selectContent?.classList.add("flex-1", "h-auto");
            uploadContent?.classList.add("hidden");
        } else if (tab === "upload") {
            // Style upload tab as active
            uploadTab?.classList.remove(
                "text-gray-500",
                "border-transparent",
                "hover:text-gray-700",
                "hover:border-gray-300",
            );
            uploadTab?.classList.add("text-gray-50", "bg-gray-700");

            // Style select tab as inactive
            selectTab?.classList.remove("text-gray-50", "bg-gray-700");
            selectTab?.classList.add(
                "text-gray-500",
                "border-transparent",
                "hover:text-gray-700",
                "hover:border-gray-300",
            );

            // Show upload content, hide select content
            uploadContent?.classList.remove("hidden");
            selectContent?.classList.add("h-0", "overflow-hidden");
            selectContent?.classList.remove("flex-1", "h-auto");

            // Load upload form if needed
            this.loadUploadForm();
        }
    }

    toggleImageSelection(element) {
        const imageId = element.getAttribute("data-image-id");
        const isSelected = this.selectedImages.has(imageId);

        if (isSelected) {
            this.deselectImageElement(element, imageId);
        } else {
            if (this.config.mode === "multiple") {
                if (this.selectedImages.size >= this.config.maxSelection) {
                    this.showMessage(
                        `Máximo ${this.config.maxSelection} imágenes permitidas`,
                        "warning",
                    );
                    return;
                }
            } else {
                this.clearSelection();
            }

            this.selectImageElement(element, imageId);
        }

        this.updateSelectionUI();
    }

    selectImageElement(element, imageId) {
        const img = element.querySelector("img");
        const imageData = {
            id: imageId,
            src: img?.src || "",
            name: img?.alt || "",
        };

        this.selectedImages.set(imageId, imageData);
        this.applySelectionToElement(element, imageId);
    }

    applySelectionToElement(element, imageId) {
        element.classList.remove("border-gray-200", "hover:border-gray-300");
        element.classList.add("border-accent", "bg-accent/10");

        const indicator = element.querySelector("[data-selection-indicator]");
        if (indicator) {
            indicator.classList.remove(
                "bg-white",
                "border-gray-300",
                "group-hover:border-gray-400",
            );
            indicator.classList.add("bg-accent", "border-accent");
            indicator.innerHTML =
                '<svg class="w-4 h-4 text-white" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path></svg>';
        }
    }

    deselectImageElement(element, imageId) {
        this.selectedImages.delete(imageId);

        element.classList.add("border-gray-200", "hover:border-gray-300");
        element.classList.remove("border-accent", "bg-accent/10");

        const indicator = element.querySelector("[data-selection-indicator]");
        if (indicator) {
            indicator.classList.add(
                "bg-white",
                "border-gray-300",
                "group-hover:border-gray-400",
            );
            indicator.classList.remove("bg-accent", "border-accent");
            indicator.innerHTML = "";
        }
    }

    clearSelection() {
        this.selectedImages.forEach((imageData, imageId) => {
            const element = this.modalElement.querySelector(
                `[data-image-id="${imageId}"]`,
            );
            if (element) {
                this.deselectImageElement(element, imageId);
            }
        });
        this.selectedImages.clear();
        this.updateSelectionUI();
    }

    removeFromSelection(imageId) {
        const element = this.modalElement.querySelector(
            `[data-image-id="${imageId}"]`,
        );
        if (element) {
            this.deselectImageElement(element, imageId);
        }
        this.updateSelectionUI();
    }

    updateSelectionUI() {
        const selectedCount = this.selectedImages.size;
        const confirmButton = this.modalElement.querySelector(
            "[data-confirm-selection]",
        );

        if (confirmButton) {
            confirmButton.disabled = selectedCount === 0;
        }

        if (this.config.mode === "multiple") {
            const countElement = this.modalElement.querySelector(
                "[data-selected-count]",
            );
            if (countElement) {
                countElement.textContent = selectedCount;
            }
        }

        this.updateSelectedImagesPreview();
    }

    updateSelectedImagesPreview() {
        const previewContainer = this.modalElement.querySelector(
            "[data-selected-preview]",
        );
        if (!previewContainer) return;

        previewContainer.innerHTML = "";

        this.selectedImages.forEach((imageData, imageId) => {
            const preview = document.createElement("div");
            preview.className =
                "relative w-12 h-12 rounded-md overflow-hidden border border-gray-200";
            preview.innerHTML = `
                <img src="${imageData.src}" alt="${imageData.name}" class="w-full h-full object-cover">
                <button data-remove-from-selection data-image-id="${imageId}" class="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white rounded-full text-xs hover:bg-red-600">×</button>
            `;
            previewContainer.appendChild(preview);

            // Add event listener for the remove button
            const removeBtn = preview.querySelector(
                "[data-remove-from-selection]",
            );
            removeBtn.addEventListener("click", () =>
                this.removeFromSelection(imageId),
            );
            this.eventListeners.push({
                element: removeBtn,
                event: "click",
                handler: () => this.removeFromSelection(imageId),
            });
        });
    }

    debounceSearch(query) {
        clearTimeout(this.searchTimeout);
        this.searchTimeout = setTimeout(() => {
            this.loadResults({ name: query, page: 1 });
        }, 300);
    }

    updateSort(sortValue) {
        this.loadResults({ sort: sortValue, page: 1 });
    }

    loadPage(page) {
        const searchInput = this.modalElement.querySelector(
            "[data-search-input]",
        );
        const sortSelect =
            this.modalElement.querySelector("[data-sort-select]");

        this.loadResults({
            name: searchInput?.value || "",
            sort: sortSelect?.value || "created_at_desc",
            page: page,
        });
    }

    loadResults(params) {
        const loading = this.modalElement.querySelector("[data-loading]");
        const grid = this.modalElement.querySelector("#images-selector-grid");

        if (loading) loading.classList.remove("hidden");

        const urlParams = new URLSearchParams(params);
        urlParams.append("limit", "12");

        // Use HTMX to load new results
        if (window.htmx && grid) {
            window.htmx.ajax("GET", `/api/images/selector?${urlParams}`, {
                target: "#images-selector-grid",
                swap: "innerHTML",
            });
        }
    }

    loadUploadForm() {
        const container = this.modalElement.querySelector(
            "#images-selector-modal-upload-container",
        );
        if (container && container.innerHTML.trim() === "") {
            if (window.htmx) {
                window.htmx.ajax("GET", "/imagenes/subir", {
                    target: "#images-selector-modal-upload-container",
                    swap: "innerHTML",
                });
            }
        }
    }

    async confirmSelection() {
        const selectedIds = Array.from(this.selectedImages.keys());

        if (selectedIds.length === 0) {
            this.showMessage("Selecciona al menos una imagen", "warning");
            return;
        }

        if (this.config.updateEndpoint) {
            try {
                await this.updateEntity(selectedIds);
                this.showMessage(
                    "Imágenes actualizadas correctamente",
                    "success",
                );
                this.close();
            } catch (error) {
                console.error("Error updating entity:", error);
                this.showMessage("Error al actualizar las imágenes", "error");
            }
        } else {
            // Fallback to form field update for backward compatibility
            this.updateFormField(selectedIds);
            this.close();
        }
    }

    async updateEntity(selectedIds) {
        if (!this.config.updateEndpoint) {
            throw new Error("No update endpoint configured");
        }

        const response = await fetch(this.config.updateEndpoint, {
            method: "PUT",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                image_ids: selectedIds,
            }),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        return response.json();
    }

    updateFormField(selectedIds) {
        const targetField = this.config.targetField;
        if (!targetField) return;

        const formField = document.querySelector(`[name="${targetField}"]`);
        if (formField) {
            if (this.config.mode === "single") {
                formField.value = selectedIds[0] || "";
            } else {
                formField.value = selectedIds.join(",");
            }

            formField.dispatchEvent(new Event("change", { bubbles: true }));
        }

        this.updateFormImagePreview(
            targetField,
            Array.from(this.selectedImages.values()),
        );
    }

    updateFormImagePreview(targetField, selectedImages) {
        const previewContainer = document.getElementById(
            `${targetField}-preview`,
        );
        if (!previewContainer) return;

        previewContainer.innerHTML = "";

        selectedImages.forEach((imageData) => {
            const preview = document.createElement("div");
            preview.className =
                "relative inline-block w-20 h-20 rounded-md overflow-hidden border border-gray-200 mr-2 mb-2";
            preview.innerHTML = `
                <img src="${imageData.src}" alt="${imageData.name}" class="w-full h-full object-cover">
                <div class="absolute bottom-0 left-0 right-0 bg-black bg-opacity-50 text-white text-xs p-1 truncate">${imageData.name}</div>
            `;
            previewContainer.appendChild(preview);
        });
    }

    showMessage(message, type = "info") {
        // Use existing toast system if available
        if (window.showToast) {
            window.showToast(message, type);
        } else {
            // Fallback to alert
            alert(message);
        }
    }

    close() {
        this.cleanup();
        if (this.modalElement) {
            this.modalElement.remove();
        }
    }

    cleanup() {
        // Clear timeouts
        if (this.searchTimeout) {
            clearTimeout(this.searchTimeout);
        }

        // Remove event listeners
        this.eventListeners.forEach(({ element, event, handler }) => {
            element.removeEventListener(event, handler);
        });
        this.eventListeners = [];

        // Clear selected images
        this.selectedImages.clear();
    }

    static create(config) {
        return new ImageSelectorManager(config);
    }
}

// Make available globally
window.ImageSelectorManager = ImageSelectorManager;

