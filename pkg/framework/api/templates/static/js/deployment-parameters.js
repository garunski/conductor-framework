// Deployment Parameters - Editor-First with Slide-Out Schema Panel
// Full-width YAML editor with on-demand schema reference

(function() {
    'use strict';
    
    let instanceData = {};
    let deployedData = {};
    let services = [];
    let crdSchema = null;
    let yamlEditor = null;
    let originalYaml = '';
    let currentFieldPath = null;
    let schemaPanelOpen = false;
    
    /**
     * Initialize the deployment parameters page
     */
    function init(instanceSpecJSON, servicesJSON) {
        // Parse instance data
        try {
            if (instanceSpecJSON && instanceSpecJSON !== '{}' && typeof instanceSpecJSON === 'string') {
                instanceData = JSON.parse(instanceSpecJSON);
            } else if (instanceSpecJSON && typeof instanceSpecJSON === 'object') {
                instanceData = instanceSpecJSON;
            } else {
                instanceData = {};
            }
            
            if (servicesJSON) {
                if (typeof servicesJSON === 'string') {
                    services = JSON.parse(servicesJSON);
                } else if (Array.isArray(servicesJSON)) {
                    services = servicesJSON;
                } else {
                    services = [];
                }
            } else {
                services = [];
            }
        } catch (e) {
            console.error('Failed to parse instance data:', e);
            showError('Failed to load parameters. Please refresh the page.');
            return;
        }
        
        // Ensure services map exists
        if (!instanceData.services) {
            instanceData.services = {};
        }
        
        // Initialize all components
        Promise.all([
            fetchSchema(),
            loadDeployedValues()
        ]).then(() => {
            initializeYamlEditor();
            initializeSchemaPanel();
            setupEventHandlers();
            setupKeyboardShortcuts();
        }).catch(error => {
            console.error('Error initializing components:', error);
            // Still try to initialize editor even if schema/deployed values fail
            initializeYamlEditor();
            setupEventHandlers();
            setupKeyboardShortcuts();
        });
    }
    
    /**
     * Fetch CRD schema from API
     */
    async function fetchSchema() {
        try {
            const response = await fetch('/api/parameters/schema');
            if (response.ok) {
                crdSchema = await response.json();
            } else {
                console.warn('Failed to fetch schema');
            }
        } catch (error) {
            console.warn('Error fetching schema:', error);
        }
    }
    
    /**
     * Load deployed values from API
     */
    async function loadDeployedValues() {
        try {
            const paramsResponse = await fetch('/api/parameters');
            if (paramsResponse.ok) {
                const paramsData = await paramsResponse.json();
                deployedData = paramsData || { global: {}, services: {} };
            }
        } catch (error) {
            console.warn('Error loading deployed values:', error);
            deployedData = instanceData;
        }
    }
    
    /**
     * Initialize YAML Editor
     */
    function initializeYamlEditor() {
        const editorContainer = document.getElementById('yaml-editor');
        if (!editorContainer) {
            console.error('Editor container not found');
            return;
        }
        
        // Check if ace is available
        if (typeof ace === 'undefined') {
            console.error('Ace editor not loaded');
            editorContainer.innerHTML = '<div class="alert alert-danger p-3">Error: Ace editor failed to load. Please refresh the page.</div>';
            return;
        }
        
        try {
            // Create Ace editor
            yamlEditor = ace.edit('yaml-editor');
            yamlEditor.setTheme('ace/theme/github');
            yamlEditor.session.setMode('ace/mode/yaml');
            yamlEditor.setOptions({
                fontSize: 14,
                showPrintMargin: false,
                wrap: true,
                tabSize: 2,
                useSoftTabs: true
            });
            
            // Enable autocompletion (requires language_tools extension)
            if (typeof ace !== 'undefined' && ace.require) {
                try {
                    ace.require("ace/ext/language_tools");
                    yamlEditor.setOptions({
                        enableBasicAutocompletion: true,
                        enableLiveAutocompletion: false
                    });
                } catch (e) {
                    console.warn('Language tools extension not available:', e);
                }
            }
            
            // Convert instance data to YAML (this is the current CRD spec state)
            let yamlContent = jsonToYaml(instanceData);
            
            // If empty, show a default structure
            if (!yamlContent || yamlContent.trim() === '') {
                yamlContent = `global:
  namespace: default
  namePrefix: ""
  replicas: 1

services:
`;
            }
            
            yamlEditor.setValue(yamlContent, -1);
            originalYaml = yamlContent;
            yamlEditor.getSession().getUndoManager().reset();
            
            // Resize editor to fit container
            setTimeout(() => {
                if (yamlEditor) {
                    yamlEditor.resize();
                }
            }, 100);
            
            // Update current state indicator
            updateCurrentStateIndicator();
            
            // Track cursor position to highlight field in schema
            yamlEditor.getSession().selection.on('changeCursor', function() {
                if (schemaPanelOpen) {
                    updateSchemaHighlight();
                }
            });
            
            // Auto-validate on change
            yamlEditor.getSession().on('change', function() {
                validateYaml();
            });
            
            // Initial validation
            validateYaml();
            
            // Resize on window resize
            window.addEventListener('resize', function() {
                if (yamlEditor) {
                    yamlEditor.resize();
                }
            });
            
        } catch (error) {
            console.error('Error initializing editor:', error);
            editorContainer.innerHTML = `<div class="alert alert-danger p-3">Error initializing editor: ${escapeHtml(error.message)}</div>`;
        }
    }
    
    /**
     * Initialize Schema Panel
     */
    function initializeSchemaPanel() {
        const container = document.getElementById('schema-explorer');
        if (!container) return;
        
        if (!crdSchema || !crdSchema.properties) {
            container.innerHTML = '<div class="text-muted small p-3">Schema not available</div>';
            return;
        }
        
        let html = '<div class="schema-tree">';
        
        // Global schema
        if (crdSchema.properties.global) {
            html += buildSchemaSection('global', 'Global', crdSchema.properties.global, 'global');
        }
        
        // Services schema
        if (crdSchema.properties.services) {
            const servicesSchema = crdSchema.properties.services;
            let serviceTemplate = null;
            
            if (servicesSchema.additionalProperties && typeof servicesSchema.additionalProperties === 'object') {
                serviceTemplate = servicesSchema.additionalProperties;
            } else if (servicesSchema.items && servicesSchema.items.properties) {
                serviceTemplate = servicesSchema.items;
            }
            
            if (serviceTemplate) {
                html += '<div class="mb-3">';
                html += '<div class="fw-bold mb-2">📁 Services</div>';
                
                services.forEach(serviceName => {
                    html += buildSchemaSection(
                        `services.${serviceName}`,
                        serviceName,
                        serviceTemplate,
                        `services.${serviceName}`
                    );
                });
                
                html += '</div>';
            }
        }
        
        html += '</div>';
        container.innerHTML = html;
        
        // Add click handlers
        container.querySelectorAll('.schema-field-item').forEach(item => {
            item.addEventListener('click', function() {
                const fieldPath = this.getAttribute('data-field-path');
                scrollToFieldInEditor(fieldPath);
            });
        });
    }
    
    /**
     * Build schema section
     */
    function buildSchemaSection(id, title, schema, prefix) {
        let html = '<div class="mb-3">';
        html += `<div class="fw-bold mb-2">📁 ${title}</div>`;
        html += buildSchemaTree(schema, 0, prefix);
        html += '</div>';
        return html;
    }
    
    /**
     * Build schema tree recursively
     */
    function buildSchemaTree(schema, level, prefix = '') {
        if (!schema || !schema.properties) return '';
        
        const indent = level * 16;
        let html = '<ul class="list-unstyled mb-0" style="margin-left: ' + indent + 'px;">';
        const properties = schema.properties || {};
        const required = schema.required || [];
        
        Object.keys(properties).forEach(key => {
            const prop = properties[key];
            const isRequired = required.includes(key);
            const fieldPath = prefix ? `${prefix}.${key}` : key;
            const fieldId = `schema-field-${fieldPath.replace(/\./g, '-').replace(/\[/g, '-').replace(/\]/g, '')}`;
            
            html += '<li class="mb-1">';
            html += `<div class="schema-field-item" data-field-path="${fieldPath}" id="${fieldId}">`;
            html += `<span class="fw-semibold">${key}</span>`;
            
            // Type badge
            if (prop.type) {
                const badgeColor = getTypeBadgeColor(prop.type);
                html += ` <span class="badge ${badgeColor} badge-sm">${prop.type}</span>`;
            }
            
            // Required indicator
            if (isRequired) {
                html += ' <span class="text-danger small">*</span>';
            }
            
            // Description
            if (prop.description) {
                html += `<div class="text-muted small mt-1">${escapeHtml(prop.description)}</div>`;
            }
            
            // Default value
            if (prop.default !== undefined) {
                html += `<div class="text-info small mt-1">Default: <code>${escapeHtml(JSON.stringify(prop.default))}</code></div>`;
            }
            
            // Enum values
            if (prop.enum && Array.isArray(prop.enum)) {
                html += `<div class="text-muted small mt-1">Allowed: ${prop.enum.map(v => `<code>${escapeHtml(String(v))}</code>`).join(', ')}</div>`;
            }
            
            // Min/Max for numbers
            if (prop.type === 'integer' || prop.type === 'number') {
                const constraints = [];
                if (prop.minimum !== undefined) constraints.push(`Min: ${prop.minimum}`);
                if (prop.maximum !== undefined) constraints.push(`Max: ${prop.maximum}`);
                if (constraints.length > 0) {
                    html += `<div class="text-muted small mt-1">${constraints.join(', ')}</div>`;
                }
            }
            
            html += '</div>';
            
            // Nested properties
            if (prop.type === 'object' && prop.properties) {
                html += buildSchemaTree(prop, level + 1, fieldPath);
            } else if (prop.type === 'array' && prop.items && prop.items.type === 'object' && prop.items.properties) {
                html += buildSchemaTree(prop.items, level + 1, `${fieldPath}[]`);
            }
            
            html += '</li>';
        });
        
        html += '</ul>';
        return html;
    }
    
    /**
     * Get badge color for type
     */
    function getTypeBadgeColor(type) {
        const colors = {
            'string': 'bg-primary',
            'integer': 'bg-info',
            'number': 'bg-info',
            'boolean': 'bg-success',
            'array': 'bg-warning',
            'object': 'bg-secondary'
        };
        return colors[type] || 'bg-secondary';
    }
    
    /**
     * Update schema highlight based on cursor position
     */
    function updateSchemaHighlight() {
        if (!yamlEditor || !schemaPanelOpen) return;
        
        const cursor = yamlEditor.getCursorPosition();
        const line = yamlEditor.session.getLine(cursor.row);
        
        // Find the field at cursor position
        const fieldPath = findFieldAtCursor(line, cursor.column);
        
        if (fieldPath && fieldPath !== currentFieldPath) {
            // Remove previous highlight
            document.querySelectorAll('.schema-field-item.selected').forEach(item => {
                item.classList.remove('selected');
            });
            
            // Add new highlight
            const fieldId = `schema-field-${fieldPath.replace(/\./g, '-').replace(/\[/g, '-').replace(/\]/g, '')}`;
            const fieldElement = document.getElementById(fieldId);
            if (fieldElement) {
                fieldElement.classList.add('selected');
                fieldElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
                currentFieldPath = fieldPath;
            }
        }
    }
    
    /**
     * Find field path at cursor position
     */
    function findFieldAtCursor(line, column) {
        // Simple heuristic: find the key before the cursor
        const beforeCursor = line.substring(0, column);
        const match = beforeCursor.match(/(\w+)\s*:\s*$/);
        if (match) {
            return match[1];
        }
        
        // Try to find nested path
        const lines = yamlEditor.session.getLines(0, yamlEditor.getCursorPosition().row);
        const path = [];
        
        for (let i = lines.length - 1; i >= 0; i--) {
            const lineMatch = lines[i].match(/^(\s*)(\w+)\s*:/);
            if (lineMatch) {
                const indent = lineMatch[1].length;
                if (indent < path.length * 2) {
                    path.length = Math.floor(indent / 2);
                }
                path.push(lineMatch[2]);
            }
        }
        
        return path.length > 0 ? path.reverse().join('.') : null;
    }
    
    /**
     * Scroll to field in editor
     */
    function scrollToFieldInEditor(fieldPath) {
        if (!yamlEditor) return;
        
        const yamlContent = yamlEditor.getValue();
        const lines = yamlContent.split('\n');
        const fieldName = fieldPath.split('.').pop();
        
        for (let i = 0; i < lines.length; i++) {
            if (lines[i].trim().startsWith(fieldName + ':')) {
                yamlEditor.gotoLine(i + 1, 0, true);
                yamlEditor.focus();
                break;
            }
        }
    }
    
    /**
     * Toggle schema panel
     */
    function toggleSchemaPanel() {
        const panel = document.getElementById('schema-panel');
        const editorContainer = document.getElementById('editor-container');
        
        if (!panel) return;
        
        schemaPanelOpen = !schemaPanelOpen;
        panel.classList.toggle('open');
        
        // Adjust editor container margin
        if (editorContainer) {
            if (schemaPanelOpen) {
                editorContainer.classList.add('editor-with-schema');
                updateSchemaHighlight();
                // Focus search after a short delay
                setTimeout(() => {
                    const search = document.getElementById('schema-search');
                    if (search) search.focus();
                }, 300);
            } else {
                editorContainer.classList.remove('editor-with-schema');
            }
        }
    }
    
    /**
     * Setup event handlers
     */
    function setupEventHandlers() {
        // Schema panel toggle
        const toggleBtn = document.getElementById('btn-schema-toggle');
        if (toggleBtn) {
            toggleBtn.addEventListener('click', toggleSchemaPanel);
        }
        
        const closeBtn = document.getElementById('btn-schema-close');
        if (closeBtn) {
            closeBtn.addEventListener('click', toggleSchemaPanel);
        }
        
        // Validate button
        const validateBtn = document.getElementById('btn-validate-yaml');
        if (validateBtn) {
            validateBtn.addEventListener('click', function() {
                validateYaml();
            });
        }
        
        // Diff button
        const diffBtn = document.getElementById('btn-diff-yaml');
        if (diffBtn) {
            diffBtn.addEventListener('click', function() {
                showDiff();
            });
        }
        
        // View deployed button
        const viewDeployedBtn = document.getElementById('btn-view-deployed');
        if (viewDeployedBtn) {
            viewDeployedBtn.addEventListener('click', function() {
                showDeployedValues();
            });
        }
        
        // Reset button
        const resetBtn = document.getElementById('btn-reset-yaml');
        if (resetBtn) {
            resetBtn.addEventListener('click', function() {
                if (confirm('Reset to original values?')) {
                    yamlEditor.setValue(originalYaml, -1);
                    yamlEditor.getSession().getUndoManager().reset();
                    validateYaml();
                }
            });
        }
        
        // Form submission
        const form = document.getElementById('parameters-form');
        if (form) {
            form.addEventListener('submit', async function(e) {
                e.preventDefault();
                await applyChanges();
            });
        }
        
        // Schema search
        const schemaSearch = document.getElementById('schema-search');
        if (schemaSearch) {
            schemaSearch.addEventListener('input', function() {
                filterSchema(this.value);
            });
        }
    }
    
    /**
     * Setup keyboard shortcuts
     */
    function setupKeyboardShortcuts() {
        document.addEventListener('keydown', function(e) {
            // Ctrl+Space or Cmd+Space: Toggle schema panel
            if ((e.ctrlKey || e.metaKey) && e.key === ' ') {
                e.preventDefault();
                toggleSchemaPanel();
            }
            
            // Ctrl+K or Cmd+K: Focus schema search
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                if (schemaPanelOpen) {
                    e.preventDefault();
                    const search = document.getElementById('schema-search');
                    if (search) search.focus();
                }
            }
            
            // Esc: Close schema panel
            if (e.key === 'Escape' && schemaPanelOpen) {
                const search = document.getElementById('schema-search');
                if (document.activeElement === search) {
                    // If search is focused, close panel
                    toggleSchemaPanel();
                } else if (search) {
                    // Otherwise, just blur search
                    search.blur();
                }
            }
        });
    }
    
    /**
     * Filter schema
     */
    function filterSchema(query) {
        const items = document.querySelectorAll('.schema-field-item');
        const lowerQuery = query.toLowerCase();
        
        items.forEach(item => {
            const text = item.textContent.toLowerCase();
            const parent = item.closest('li');
            if (parent) {
                if (text.includes(lowerQuery) || query === '') {
                    parent.style.display = '';
                    // Expand parent sections
                    let current = parent.parentElement;
                    while (current && current.tagName !== 'BODY') {
                        if (current.classList && current.classList.contains('collapse')) {
                            const bsCollapse = bootstrap.Collapse.getInstance(current);
                            if (bsCollapse) bsCollapse.show();
                        }
                        current = current.parentElement;
                    }
                } else {
                    parent.style.display = 'none';
                }
            }
        });
    }
    
    /**
     * Convert JSON to YAML
     */
    function jsonToYaml(obj) {
        if (!obj || Object.keys(obj).length === 0) {
            return '';
        }
        
        try {
            if (typeof jsyaml !== 'undefined') {
                return jsyaml.dump(obj, {
                    indent: 2,
                    lineWidth: -1,
                    noRefs: true,
                    sortKeys: false
                });
            } else {
                return JSON.stringify(obj, null, 2);
            }
        } catch (e) {
            console.error('Error converting JSON to YAML:', e);
            return '';
        }
    }
    
    /**
     * Convert YAML to JSON
     */
    function yamlToJson(yamlStr) {
        if (!yamlStr || yamlStr.trim() === '') {
            return {};
        }
        
        try {
            if (typeof jsyaml !== 'undefined') {
                return jsyaml.load(yamlStr) || {};
            } else {
                return JSON.parse(yamlStr);
            }
        } catch (e) {
            throw new Error(`Invalid YAML: ${e.message}`);
        }
    }
    
    /**
     * Validate YAML
     */
    function validateYaml() {
        const statusDiv = document.getElementById('yaml-validation-status');
        if (!statusDiv || !yamlEditor) return;
        
        const yamlContent = yamlEditor.getValue();
        
        try {
            const data = yamlToJson(yamlContent);
            statusDiv.innerHTML = '<span class="text-success">✓ Valid YAML</span>';
            statusDiv.className = 'small text-success';
            return true;
        } catch (e) {
            statusDiv.innerHTML = `<span class="text-danger">✗ ${escapeHtml(e.message)}</span>`;
            statusDiv.className = 'small text-danger';
            return false;
        }
    }
    
    /**
     * Update current state indicator
     */
    function updateCurrentStateIndicator() {
        const indicator = document.getElementById('current-state-indicator');
        if (!indicator) return;
        
        const hasData = instanceData && (
            (instanceData.global && Object.keys(instanceData.global).length > 0) ||
            (instanceData.services && Object.keys(instanceData.services).length > 0)
        );
        
        if (hasData) {
            indicator.innerHTML = '<span class="text-success">✓</span> Showing current CRD spec state';
        } else {
            indicator.innerHTML = '<span class="text-muted">No parameters configured</span>';
        }
    }
    
    /**
     * Show deployed values in a modal
     */
    function showDeployedValues() {
        if (!deployedData || (Object.keys(deployedData).length === 0 && (!deployedData.global || Object.keys(deployedData.global).length === 0))) {
            alert('No deployed values available. The current editor shows the CRD spec state.');
            return;
        }
        
        const deployedYaml = jsonToYaml(deployedData);
        const modal = document.getElementById('confirm-modal');
        const modalTitle = document.getElementById('confirm-modal-label');
        const modalBody = document.getElementById('confirm-modal-message');
        const modalFooter = document.querySelector('#confirm-modal .modal-footer');
        
        if (!modal || !modalTitle || !modalBody) return;
        
        modalTitle.textContent = 'Deployed Values';
        modalBody.innerHTML = `
            <div class="mb-3">
                <small class="text-muted">These are the values currently deployed in the cluster:</small>
            </div>
            <pre style="background: #f8f9fa; padding: 1rem; border-radius: 4px; max-height: 400px; overflow-y: auto; font-size: 12px;"><code>${escapeHtml(deployedYaml)}</code></pre>
            <div class="mt-3">
                <button type="button" class="btn btn-sm btn-outline-primary" id="btn-load-deployed">Load into Editor</button>
            </div>
        `;
        
        // Hide default footer buttons
        if (modalFooter) {
            modalFooter.style.display = 'none';
        }
        
        // Show modal
        const bsModal = new bootstrap.Modal(modal);
        bsModal.show();
        
        // Handle load button
        const loadBtn = document.getElementById('btn-load-deployed');
        if (loadBtn) {
            loadBtn.addEventListener('click', function() {
                if (confirm('Load deployed values into editor? This will replace current content.')) {
                    yamlEditor.setValue(deployedYaml, -1);
                    instanceData = deployedData;
                    originalYaml = deployedYaml;
                    yamlEditor.getSession().getUndoManager().reset();
                    updateCurrentStateIndicator();
                    validateYaml();
                    bsModal.hide();
                }
            });
        }
        
        // Restore footer on hide
        modal.addEventListener('hidden.bs.modal', function() {
            if (modalFooter) {
                modalFooter.style.display = '';
            }
            modalBody.innerHTML = '';
        }, { once: true });
    }
    
    /**
     * Show diff between current and deployed
     */
    function showDiff() {
        const diffDiv = document.getElementById('yaml-diff-summary');
        if (!diffDiv || !yamlEditor) return;
        
        try {
            const currentData = yamlToJson(yamlEditor.getValue());
            const changes = calculateDiff(deployedData, currentData);
            
            if (changes.length === 0) {
                diffDiv.innerHTML = '<span class="text-success">No changes</span>';
            } else {
                diffDiv.innerHTML = `<span class="text-warning">${changes.length} change(s) detected</span>`;
                // Show detailed diff in a modal or expandable section
                console.log('Changes:', changes);
            }
        } catch (e) {
            diffDiv.innerHTML = '<span class="text-danger">Error calculating diff</span>';
        }
    }
    
    /**
     * Calculate diff between two objects
     */
    function calculateDiff(oldObj, newObj, path = '') {
        const changes = [];
        
        if (!oldObj) oldObj = {};
        if (!newObj) newObj = {};
        
        // Check for new/modified keys
        Object.keys(newObj).forEach(key => {
            const newPath = path ? `${path}.${key}` : key;
            const oldVal = getNestedValue(oldObj, key);
            const newVal = newObj[key];
            
            if (oldVal === undefined) {
                changes.push({ path: newPath, type: 'added', value: newVal });
            } else if (JSON.stringify(oldVal) !== JSON.stringify(newVal)) {
                if (typeof newVal === 'object' && newVal !== null && !Array.isArray(newVal) && 
                    typeof oldVal === 'object' && oldVal !== null && !Array.isArray(oldVal)) {
                    changes.push(...calculateDiff(oldVal, newVal, newPath));
                } else {
                    changes.push({ path: newPath, type: 'modified', oldValue: oldVal, newValue: newVal });
                }
            } else if (typeof newVal === 'object' && newVal !== null && !Array.isArray(newVal)) {
                changes.push(...calculateDiff(oldVal || {}, newVal, newPath));
            }
        });
        
        // Check for removed keys
        Object.keys(oldObj).forEach(key => {
            if (newObj[key] === undefined) {
                const oldPath = path ? `${path}.${key}` : key;
                changes.push({ path: oldPath, type: 'removed', value: oldObj[key] });
            }
        });
        
        return changes;
    }
    
    /**
     * Get nested value from object
     */
    function getNestedValue(obj, path) {
        const keys = path.split('.');
        let current = obj;
        for (const key of keys) {
            if (current === undefined || current === null) return undefined;
            current = current[key];
        }
        return current;
    }
    
    /**
     * Apply changes to cluster
     */
    async function applyChanges() {
        const applyBtn = document.getElementById('btn-apply-yaml');
        if (applyBtn) {
            setButtonLoading(applyBtn, true);
        }
        
        try {
            // Validate first
            if (!validateYaml()) {
                throw new Error('Invalid YAML. Please fix errors before applying.');
            }
            
            const yamlContent = yamlEditor.getValue();
            const data = yamlToJson(yamlContent);
            
            // Show diff in confirmation
            const changes = calculateDiff(deployedData, data);
            if (changes.length > 0) {
                const changeSummary = changes.slice(0, 5).map(c => {
                    if (c.type === 'modified') {
                        return `${c.path}: ${JSON.stringify(c.oldValue)} → ${JSON.stringify(c.newValue)}`;
                    } else if (c.type === 'added') {
                        return `${c.path}: +${JSON.stringify(c.value)}`;
                    } else {
                        return `${c.path}: -${JSON.stringify(c.value)}`;
                    }
                }).join('\n');
                
                const fullMessage = `Apply ${changes.length} change(s)?\n\n${changeSummary}${changes.length > 5 ? '\n...' : ''}`;
                if (!confirm(fullMessage)) {
                    return;
                }
            }
            
            // Submit to API
            const response = await fetch('/api/parameters', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });
            
            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.message || 'Failed to save parameters');
            }
            
            showStatus('Parameters applied successfully', 'success');
            originalYaml = yamlContent;
            instanceData = data;
            
            // Reload deployed values and update indicator
            await loadDeployedValues();
            updateCurrentStateIndicator();
            
        } catch (error) {
            console.error('Error applying changes:', error);
            showStatus(`Error: ${error.message}`, 'error');
        } finally {
            if (applyBtn) {
                setButtonLoading(applyBtn, false);
            }
        }
    }
    
    /**
     * Show status message
     */
    function showStatus(message, type) {
        const statusDiv = document.getElementById('parameters-status');
        if (!statusDiv) return;
        
        let alertClass = 'alert-info';
        if (type === 'success') {
            alertClass = 'alert-success';
        } else if (type === 'error') {
            alertClass = 'alert-danger';
        }
        
        statusDiv.innerHTML = `<div class="alert ${alertClass} mb-0">${escapeHtml(message)}</div>`;
        
        if (type !== 'error') {
            setTimeout(() => {
                statusDiv.innerHTML = '';
            }, 5000);
        }
    }
    
    /**
     * Show error message
     */
    function showError(message) {
        showStatus(message, 'error');
    }
    
    /**
     * Escape HTML
     */
    function escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
    
    /**
     * Set button loading state
     */
    function setButtonLoading(button, loading) {
        if (!button) return;
        if (loading) {
            button.disabled = true;
            button.setAttribute('aria-busy', 'true');
            button.classList.add('opacity-75');
        } else {
            button.disabled = false;
            button.removeAttribute('aria-busy');
            button.classList.remove('opacity-75');
        }
    }
    
    // Expose initialization function
    window.deploymentParams = {
        init
    };
})();
