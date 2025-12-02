// YAML Editor management (Ace Editor)

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    const State = DeploymentParams.State;
    const YamlUtils = DeploymentParams.YamlUtils;
    
    DeploymentParams.YamlEditor = {
        initialize: function() {
            const editorContainer = document.getElementById('yaml-editor');
            if (!editorContainer) {
                return;
            }
            
            const parentContainer = document.getElementById('yaml-editor-container');
            if (parentContainer) {
                parentContainer.classList.add('active');
            }
            
            if (typeof ace === 'undefined') {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'alert alert-danger p-3';
                errorDiv.textContent = 'Error: Ace editor failed to load. Please refresh the page.';
                editorContainer.textContent = '';
                editorContainer.appendChild(errorDiv);
                return;
            }
            
            try {
                const yamlEditor = ace.edit('yaml-editor');
                
                // Helper function to get theme based on current theme
                function getAceTheme() {
                    const theme = document.documentElement.getAttribute('data-bs-theme');
                    return theme === 'dark' ? 'ace/theme/clouds_midnight' : 'ace/theme/clouds';
                }
                
                // Use a base theme and override with custom CSS
                yamlEditor.setTheme(getAceTheme());
                yamlEditor.session.setMode('ace/mode/yaml');
                yamlEditor.setOptions({
                    fontSize: 14,
                    showPrintMargin: false,
                    wrap: true,
                    tabSize: 2,
                    useSoftTabs: true,
                    fontFamily: "'Courier New', 'Monaco', 'Menlo', 'Consolas', monospace"
                });
                
                // Listen for theme changes
                const themeObserver = new MutationObserver(function(mutations) {
                    mutations.forEach(function(mutation) {
                        if (mutation.type === 'attributes' && mutation.attributeName === 'data-bs-theme') {
                            yamlEditor.setTheme(getAceTheme());
                        }
                    });
                });
                
                themeObserver.observe(document.documentElement, {
                    attributes: true,
                    attributeFilter: ['data-bs-theme']
                });
                
                if (typeof ace !== 'undefined' && ace.require) {
                    try {
                        ace.require("ace/ext/language_tools");
                        yamlEditor.setOptions({
                            enableBasicAutocompletion: true,
                            enableLiveAutocompletion: false
                        });
                    } catch (e) {
                        // Language tools extension not available
                    }
                }
                
                const instanceData = State.getInstanceData();
                let yamlContent = YamlUtils.jsonToYaml(instanceData);
                
                if (!yamlContent || yamlContent.trim() === '') {
                    yamlContent = `global:
  namespace: default
  namePrefix: ""
  replicas: 1

services:
`;
                }
                
                yamlEditor.setValue(yamlContent, -1);
                State.setOriginalYaml(yamlContent);
                State.setYamlEditor(yamlEditor);
                yamlEditor.getSession().getUndoManager().reset();
                
                if (parentContainer) {
                    parentContainer.classList.remove('active');
                }
                
                setTimeout(() => {
                    if (yamlEditor) {
                        yamlEditor.resize();
                    }
                }, 100);
                
                const resizeHandler = function() {
                    if (yamlEditor) {
                        yamlEditor.resize();
                    }
                };
                State.addEventListener(window, 'resize', resizeHandler);
                
            } catch (error) {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'alert alert-danger p-3';
                errorDiv.textContent = `Error initializing editor: ${error.message}`;
                editorContainer.textContent = '';
                editorContainer.appendChild(errorDiv);
            }
        },
        
        validate: function() {
            const statusDiv = document.getElementById('parameters-status');
            const yamlEditor = State.getYamlEditor();
            
            if (!statusDiv || !yamlEditor) {
                if (statusDiv) {
                    statusDiv.textContent = 'Switch to YAML editor to validate';
                    statusDiv.className = 'small text-muted';
                }
                return false;
            }
            
            const yamlContent = yamlEditor.getValue();
            
            try {
                YamlUtils.yamlToJson(yamlContent);
                statusDiv.textContent = 'VALID';
                statusDiv.className = 'small status-valid';
                return true;
            } catch (e) {
                statusDiv.textContent = e.message;
                statusDiv.className = 'small status-error';
                return false;
            }
        },
        
        scrollToField: function(fieldPath) {
            const yamlEditor = State.getYamlEditor();
            if (!yamlEditor) return;
            
            const yamlContent = yamlEditor.getValue();
            const lines = yamlContent.split('\n');
            const pathParts = fieldPath.split('.');
            const fieldName = pathParts[pathParts.length - 1];
            const expectedIndent = (pathParts.length - 1) * 2;
            
            for (let i = 0; i < lines.length; i++) {
                const line = lines[i];
                const trimmed = line.trim();
                
                if (!trimmed || trimmed.startsWith('#')) continue;
                
                if (trimmed.startsWith(fieldName + ':')) {
                    const currentIndent = line.length - line.trimStart().length;
                    if (currentIndent === expectedIndent) {
                        yamlEditor.gotoLine(i + 1, 0, true);
                        yamlEditor.focus();
                        return;
                    }
                }
            }
            
            // Fallback: just find any line with the field name
            for (let i = 0; i < lines.length; i++) {
                if (lines[i].trim().startsWith(fieldName + ':')) {
                    yamlEditor.gotoLine(i + 1, 0, true);
                    yamlEditor.focus();
                    break;
                }
            }
        }
    };
})();

