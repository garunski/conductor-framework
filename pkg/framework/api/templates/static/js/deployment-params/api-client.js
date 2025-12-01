// API client for deployment parameters

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    const State = DeploymentParams.State;
    const Utils = DeploymentParams.Utils;
    const YamlUtils = DeploymentParams.YamlUtils;
    const YamlEditor = DeploymentParams.YamlEditor;
    const ViewManager = DeploymentParams.ViewManager;
    const FieldRenderer = DeploymentParams.FieldRenderer;
    
    DeploymentParams.ApiClient = {
        fetchSchema: async function() {
            try {
                const response = await fetch('/api/parameters/schema');
                if (response.ok) {
                    const schema = await response.json();
                    // Debug: log schema structure to verify descriptions are present
                    console.debug('Loaded CRD schema:', schema);
                    if (schema.properties && schema.properties.global && schema.properties.global.properties) {
                        const namespaceProp = schema.properties.global.properties.namespace;
                        console.debug('Namespace field schema:', namespaceProp);
                        console.debug('Namespace description:', namespaceProp?.description);
                    }
                    State.setCrdSchema(schema);
                }
            } catch (error) {
                // Silently fail - schema is optional
                console.error('Failed to fetch schema:', error);
            }
        },
        
        loadDeployedValues: async function() {
            try {
                const paramsResponse = await fetch('/api/parameters');
                if (paramsResponse.ok) {
                    const paramsData = await paramsResponse.json();
                    State.setDeployedData(paramsData || { global: {}, services: {} });
                }
            } catch (error) {
                State.setDeployedData(State.getInstanceData());
            }
        },
        
        applyChanges: async function() {
            const applyBtn = document.getElementById('btn-apply-yaml');
            if (applyBtn) {
                Utils.setButtonLoading(applyBtn, true);
            }
            
            try {
                if (!YamlEditor.validate()) {
                    throw new Error('Invalid YAML. Please fix errors before applying.');
                }
                
                const yamlEditor = State.getYamlEditor();
                const yamlContent = yamlEditor.getValue();
                const data = YamlUtils.yamlToJson(yamlContent);
                
                const changes = this.calculateDiff(State.getDeployedData(), data);
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
                
                Utils.showStatus('Parameters applied successfully', 'success');
                State.setOriginalYaml(yamlContent);
                State.setInstanceData(data);
                
                await this.loadDeployedValues();
                ViewManager.updateCurrentStateIndicator();
                FieldRenderer.renderAll();
                
            } catch (error) {
                Utils.showStatus(`Error: ${error.message}`, 'error');
            } finally {
                if (applyBtn) {
                    Utils.setButtonLoading(applyBtn, false);
                }
            }
        },
        
        calculateDiff: function(oldObj, newObj, path = '') {
            const changes = [];
            
            if (!oldObj) oldObj = {};
            if (!newObj) newObj = {};
            
            Object.keys(newObj).forEach(key => {
                const newPath = path ? `${path}.${key}` : key;
                const oldVal = this.getNestedValue(oldObj, key);
                const newVal = newObj[key];
                
                if (oldVal === undefined) {
                    changes.push({ path: newPath, type: 'added', value: newVal });
                } else if (JSON.stringify(oldVal) !== JSON.stringify(newVal)) {
                    if (typeof newVal === 'object' && newVal !== null && !Array.isArray(newVal) && 
                        typeof oldVal === 'object' && oldVal !== null && !Array.isArray(oldVal)) {
                        changes.push(...this.calculateDiff(oldVal, newVal, newPath));
                    } else {
                        changes.push({ path: newPath, type: 'modified', oldValue: oldVal, newValue: newVal });
                    }
                } else if (typeof newVal === 'object' && newVal !== null && !Array.isArray(newVal)) {
                    changes.push(...this.calculateDiff(oldVal || {}, newVal, newPath));
                }
            });
            
            Object.keys(oldObj).forEach(key => {
                if (newObj[key] === undefined) {
                    const oldPath = path ? `${path}.${key}` : key;
                    changes.push({ path: oldPath, type: 'removed', value: oldObj[key] });
                }
            });
            
            return changes;
        },
        
        getNestedValue: function(obj, path) {
            const keys = path.split('.');
            let current = obj;
            for (const key of keys) {
                if (current === undefined || current === null) return undefined;
                current = current[key];
            }
            return current;
        },
        
        showDeployedValues: function() {
            const deployedData = State.getDeployedData();
            if (!deployedData || (Object.keys(deployedData).length === 0 && (!deployedData.global || Object.keys(deployedData.global).length === 0))) {
                alert('No deployed values available. The current editor shows the CRD spec state.');
                return;
            }
            
            const deployedYaml = YamlUtils.jsonToYaml(deployedData);
            const modal = document.getElementById('confirm-modal');
            const modalTitle = document.getElementById('confirm-modal-label');
            const modalBody = document.getElementById('confirm-modal-message');
            const modalFooter = document.querySelector('#confirm-modal .modal-footer');
            
            if (!modal || !modalTitle || !modalBody) return;
            
            modalTitle.textContent = 'Deployed Values';
            modalBody.textContent = '';
            
            const infoDiv = document.createElement('div');
            infoDiv.className = 'mb-3';
            const infoSmall = document.createElement('small');
            infoSmall.className = 'text-muted';
            infoSmall.textContent = 'These are the values currently deployed in the cluster:';
            infoDiv.appendChild(infoSmall);
            modalBody.appendChild(infoDiv);
            
            const pre = document.createElement('pre');
            pre.style.cssText = 'background: var(--bg); padding: 1rem; border: 2px solid var(--accent); max-height: 400px; overflow-y: auto; font-size: 12px;';
            const code = document.createElement('code');
            code.textContent = deployedYaml;
            pre.appendChild(code);
            modalBody.appendChild(pre);
            
            const buttonDiv = document.createElement('div');
            buttonDiv.className = 'mt-3 d-flex gap-2';
            const loadBtn = document.createElement('button');
            loadBtn.type = 'button';
            loadBtn.className = 'btn btn-sm btn-secondary';
            loadBtn.id = 'btn-load-deployed';
            loadBtn.textContent = 'Load into Editor';
            buttonDiv.appendChild(loadBtn);
            
            if (modalFooter) {
                modalFooter.style.display = 'none';
            }
            
            const bsModal = new bootstrap.Modal(modal);
            
            const closeBtn = document.createElement('button');
            closeBtn.type = 'button';
            closeBtn.className = 'btn btn-sm';
            closeBtn.textContent = 'Close';
            closeBtn.addEventListener('click', function() {
                bsModal.hide();
            });
            buttonDiv.appendChild(closeBtn);
            modalBody.appendChild(buttonDiv);
            
            bsModal.show();
            
            const loadHandler = function() {
                if (confirm('Load deployed values into editor? This will replace current content.')) {
                    const yamlEditor = State.getYamlEditor();
                    yamlEditor.setValue(deployedYaml, -1);
                    State.setInstanceData(deployedData);
                    State.setOriginalYaml(deployedYaml);
                    yamlEditor.getSession().getUndoManager().reset();
                    ViewManager.updateCurrentStateIndicator();
                    YamlEditor.validate();
                    bsModal.hide();
                }
            };
            State.addEventListener(loadBtn, 'click', loadHandler);
            
            const restoreHandler = function() {
                if (modalFooter) {
                    modalFooter.style.display = '';
                }
                modalBody.textContent = '';
            };
            modal.addEventListener('hidden.bs.modal', restoreHandler, { once: true });
        }
    };
})();

