// Deployment Parameters - Main initialization and orchestration

(function() {
    'use strict';
    
    const State = DeploymentParams.State;
    const Utils = DeploymentParams.Utils;
    const ApiClient = DeploymentParams.ApiClient;
    const SchemaManager = DeploymentParams.SchemaManager;
    const YamlEditor = DeploymentParams.YamlEditor;
    const ViewManager = DeploymentParams.ViewManager;
    const FieldRenderer = DeploymentParams.FieldRenderer;
    
    /**
     * Initialize the deployment parameters page
     */
    function init(instanceSpecJSON, servicesJSON) {
        // Cleanup any existing listeners
        State.cleanupEventListeners();
        
        // Parse instance data
        try {
            if (instanceSpecJSON && instanceSpecJSON !== '{}' && typeof instanceSpecJSON === 'string') {
                State.setInstanceData(JSON.parse(instanceSpecJSON));
            } else if (instanceSpecJSON && typeof instanceSpecJSON === 'object') {
                State.setInstanceData(instanceSpecJSON);
            } else {
                State.setInstanceData({});
            }
            
            if (servicesJSON) {
                if (typeof servicesJSON === 'string') {
                    State.setServices(JSON.parse(servicesJSON));
                } else if (Array.isArray(servicesJSON)) {
                    State.setServices(servicesJSON);
                } else {
                    State.setServices([]);
                }
            } else {
                State.setServices([]);
            }
        } catch (e) {
            Utils.showError('Failed to load parameters. Please refresh the page.');
            return;
        }
        
        // Ensure services map exists
        const instanceData = State.getInstanceData();
        if (!instanceData.services) {
            instanceData.services = {};
            State.setInstanceData(instanceData);
        }
        
        // Initialize all components
        Promise.all([
            ApiClient.fetchSchema(),
            ApiClient.loadDeployedValues()
        ]).then(() => {
            SchemaManager.extractTemplates();
            YamlEditor.initialize();
            FieldRenderer.renderAll();
            setupEventHandlers();
            ViewManager.updateCurrentStateIndicator();
            // Initialize toggle state - fields view is active by default
            const fieldsToggle = document.getElementById('view-toggle-fields');
            const yamlToggle = document.getElementById('view-toggle-yaml');
            if (fieldsToggle) {
                fieldsToggle.setAttribute('aria-pressed', 'true');
            }
            if (yamlToggle) {
                yamlToggle.setAttribute('aria-pressed', 'false');
            }
            // Initialize filter checkbox state - Custom Only is checked by default (show only custom fields)
            const filterCheckbox = document.getElementById('filter-custom-only');
            if (filterCheckbox) {
                filterCheckbox.checked = true;
            }
            State.setFilterNonDefaultK8s(true);
        }).catch(error => {
            // Still try to initialize even if schema/deployed values fail
            YamlEditor.initialize();
            setupEventHandlers();
            // Initialize toggle state
            const fieldsToggle = document.getElementById('view-toggle-fields');
            const yamlToggle = document.getElementById('view-toggle-yaml');
            if (fieldsToggle) {
                fieldsToggle.setAttribute('aria-pressed', 'true');
            }
            if (yamlToggle) {
                yamlToggle.setAttribute('aria-pressed', 'false');
            }
            // Initialize filter checkbox state - Custom Only is checked by default (show only custom fields)
            const filterCheckbox = document.getElementById('filter-custom-only');
            if (filterCheckbox) {
                filterCheckbox.checked = true;
            }
            State.setFilterNonDefaultK8s(true);
        });
    }
    
    /**
     * Setup event handlers
     */
    function setupEventHandlers() {
        // View toggle selector
        const fieldsToggle = document.getElementById('view-toggle-fields');
        const yamlToggle = document.getElementById('view-toggle-yaml');
        
        if (fieldsToggle) {
            const fieldsHandler = function() {
                ViewManager.showConfigurableFields();
            };
            State.addEventListener(fieldsToggle, 'click', fieldsHandler);
        }
        
        if (yamlToggle) {
            const yamlHandler = function() {
                ViewManager.showYamlEditor();
            };
            State.addEventListener(yamlToggle, 'click', yamlHandler);
        }
        
        // Validate button
        const validateBtn = document.getElementById('btn-validate-yaml');
        if (validateBtn) {
            const validateHandler = function() {
                const editorContainer = document.getElementById('yaml-editor-container');
                if (editorContainer && !editorContainer.classList.contains('active')) {
                    ViewManager.showYamlEditor();
                }
                YamlEditor.validate();
            };
            State.addEventListener(validateBtn, 'click', validateHandler);
        }
        
        // View deployed button
        const viewDeployedBtn = document.getElementById('btn-view-deployed');
        if (viewDeployedBtn) {
            State.addEventListener(viewDeployedBtn, 'click', ApiClient.showDeployedValues);
        }
        
        // Filter checkbox (Custom Only)
        const filterCheckbox = document.getElementById('filter-custom-only');
        if (filterCheckbox) {
            const filterHandler = function() {
                const showCustomOnly = this.checked;
                State.setFilterNonDefaultK8s(showCustomOnly);
                // Only re-render if we're in fields view
                const fieldsContainer = document.getElementById('configurable-fields-container');
                const editorContainer = document.getElementById('yaml-editor-container');
                const isFieldsView = fieldsContainer && !fieldsContainer.classList.contains('hidden') && 
                                    (!editorContainer || !editorContainer.classList.contains('active'));
                if (isFieldsView) {
                    FieldRenderer.renderAll();
                }
            };
            State.addEventListener(filterCheckbox, 'change', filterHandler);
        }
        
        // Reset button
        const resetBtn = document.getElementById('btn-reset-yaml');
        if (resetBtn) {
            const resetHandler = function() {
                if (confirm('Reset to original values?')) {
                    const yamlEditor = State.getYamlEditor();
                    yamlEditor.setValue(State.getOriginalYaml(), -1);
                    yamlEditor.getSession().getUndoManager().reset();
                    YamlEditor.validate();
                }
            };
            State.addEventListener(resetBtn, 'click', resetHandler);
        }
        
        // Form submission
        const form = document.getElementById('parameters-form');
        if (form) {
            const submitHandler = async function(e) {
                e.preventDefault();
                await ApiClient.applyChanges();
            };
            State.addEventListener(form, 'submit', submitHandler);
        }
        
        // Update configurable fields view when YAML changes
        const yamlEditor = State.getYamlEditor();
        if (yamlEditor) {
            yamlEditor.getSession().on('change', function() {
                try {
                    const yamlContent = yamlEditor.getValue();
                    const YamlUtils = DeploymentParams.YamlUtils;
                    State.setInstanceData(YamlUtils.yamlToJson(yamlContent));
                } catch (e) {
                    // Ignore parse errors during editing
                }
            });
        }
    }
    
    // Expose initialization function
    window.deploymentParams = {
        init
    };
})();
