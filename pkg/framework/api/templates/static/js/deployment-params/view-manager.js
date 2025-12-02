// View management (switching between YAML editor and fields view)

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    const State = DeploymentParams.State;
    
    DeploymentParams.ViewManager = {
        showYamlEditor: function() {
            const editorContainer = document.getElementById('yaml-editor-container');
            const fieldsContainer = document.getElementById('configurable-fields-container');
            
            if (editorContainer) editorContainer.classList.add('active');
            if (fieldsContainer) fieldsContainer.classList.add('hidden');
            
            // Update toggle selector
            const fieldsToggle = document.getElementById('view-toggle-fields');
            const yamlToggle = document.getElementById('view-toggle-yaml');
            if (fieldsToggle) {
                fieldsToggle.setAttribute('aria-pressed', 'false');
            }
            if (yamlToggle) {
                yamlToggle.setAttribute('aria-pressed', 'true');
            }
            
            const yamlEditor = State.getYamlEditor();
            if (yamlEditor) {
                setTimeout(() => yamlEditor.resize(), 100);
            }
        },
        
        showConfigurableFields: function() {
            const editorContainer = document.getElementById('yaml-editor-container');
            const fieldsContainer = document.getElementById('configurable-fields-container');
            
            if (editorContainer) editorContainer.classList.remove('active');
            if (fieldsContainer) fieldsContainer.classList.remove('hidden');
            
            // Update toggle selector
            const fieldsToggle = document.getElementById('view-toggle-fields');
            const yamlToggle = document.getElementById('view-toggle-yaml');
            if (fieldsToggle) {
                fieldsToggle.setAttribute('aria-pressed', 'true');
            }
            if (yamlToggle) {
                yamlToggle.setAttribute('aria-pressed', 'false');
            }
            
            DeploymentParams.FieldRenderer.renderAll();
        },
        
    };
})();

