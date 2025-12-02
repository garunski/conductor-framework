// Field rendering logic

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    const State = DeploymentParams.State;
    const SchemaManager = DeploymentParams.SchemaManager;
    const ViewManager = DeploymentParams.ViewManager;
    const YamlEditor = DeploymentParams.YamlEditor;
    const YamlUtils = DeploymentParams.YamlUtils;
    
    DeploymentParams.FieldRenderer = {
        // List of default Kubernetes parameter field names
        defaultK8sFields: new Set([
            'namespace',
            'namePrefix',
            'imageRegistry',
            'imagePullSecrets',
            'storageClassName',
            'keepPVC',
            'replicas',
            'image',
            'imageTag',
            'port',
            'resources',
            'livenessProbe',
            'readinessProbe',
            'service',
            'ingress',
            'persistence'
        ]),
        
        isDefaultK8sField: function(fieldName) {
            return this.defaultK8sFields.has(fieldName);
        },
        
        filterFields: function(fields, filterNonDefault) {
            if (!filterNonDefault) {
                return fields;
            }
            
            const filtered = {};
            Object.keys(fields).forEach(key => {
                const field = fields[key];
                const fieldName = key;
                
                // Check if this is a default k8s field
                if (this.isDefaultK8sField(fieldName)) {
                    return; // Skip default k8s fields
                }
                
                // Include the field, but also filter nested fields if they exist
                filtered[key] = { ...field };
                if (field.nested && Object.keys(field.nested).length > 0) {
                    filtered[key].nested = this.filterFields(field.nested, filterNonDefault);
                }
            });
            
            return filtered;
        },
        
        renderAll: function() {
            const container = document.getElementById('configurable-fields-content');
            if (!container) return;
            
            const crdSchema = State.getCrdSchema();
            if (!crdSchema || !crdSchema.properties) {
                const alert = document.createElement('div');
                alert.className = 'alert alert-warning';
                alert.textContent = 'Schema not available';
                container.textContent = '';
                container.appendChild(alert);
                return;
            }
            
            container.textContent = '';
            
            const filterNonDefault = State.getFilterNonDefaultK8s();
            
            const globalSchema = State.getGlobalSchema();
            if (globalSchema) {
                const instanceData = State.getInstanceData();
                const globalData = instanceData.global || {};
                const mergedGlobal = SchemaManager.mergeSchemaWithData(globalSchema, globalData, 'global');
                const filteredGlobal = this.filterFields(mergedGlobal, filterNonDefault);
                if (Object.keys(filteredGlobal).length > 0) {
                    const globalGroup = this.renderFieldGroup('Global Configuration', 'global', filteredGlobal);
                    container.appendChild(globalGroup);
                }
            }
            
            const serviceSchemaTemplate = State.getServiceSchemaTemplate();
            const services = State.getServices();
            if (serviceSchemaTemplate && services.length > 0) {
                const instanceData = State.getInstanceData();
                services.forEach(serviceName => {
                    const serviceData = (instanceData.services && instanceData.services[serviceName]) || {};
                    const mergedService = SchemaManager.mergeSchemaWithData(serviceSchemaTemplate, serviceData, `services.${serviceName}`);
                    const filteredService = this.filterFields(mergedService, filterNonDefault);
                    if (Object.keys(filteredService).length > 0) {
                        const serviceGroup = this.renderFieldGroup(serviceName, `services.${serviceName}`, filteredService);
                        container.appendChild(serviceGroup);
                    }
                });
            }
            
            if (container.children.length === 0) {
                const alert = document.createElement('div');
                alert.className = 'alert alert-info';
                alert.textContent = filterNonDefault 
                    ? 'No non-default Kubernetes parameters available' 
                    : 'No configurable fields available';
                container.appendChild(alert);
            }
            
            this.setupClickHandlers();
        },
        
        renderFieldGroup: function(title, groupPath, mergedFields) {
            const groupDiv = document.createElement('div');
            groupDiv.className = 'config-field-group';
            groupDiv.setAttribute('data-group-path', groupPath);
            
            const header = document.createElement('div');
            header.className = 'config-field-group-header';
            header.textContent = title.toUpperCase();
            groupDiv.appendChild(header);
            
            const body = document.createElement('div');
            body.className = 'config-field-group-body';
            
            const fieldKeys = Object.keys(mergedFields);
            fieldKeys.forEach((key, index) => {
                const field = mergedFields[key];
                const fieldElement = this.renderField(field, 0);
                body.appendChild(fieldElement);
                
                // Add horizontal rule between fields (except last)
                if (index < fieldKeys.length - 1) {
                    const rule = document.createElement('hr');
                    rule.className = 'config-field-rule';
                    body.appendChild(rule);
                }
            });
            
            groupDiv.appendChild(body);
            return groupDiv;
        },
        
        renderField: function(field, level) {
            const isConfigured = field.isConfigured;
            const fieldClass = isConfigured ? 'configured' : 'unconfigured';
            const dataPath = field.path;
            
            const fieldDiv = document.createElement('div');
            fieldDiv.className = `config-field-item ${fieldClass}`;
            fieldDiv.setAttribute('data-field-path', dataPath);
            if (level > 0) {
                fieldDiv.setAttribute('data-level', level.toString());
            }
            
            // Field label with type tag in neo-brutalist style
            const labelDiv = document.createElement('div');
            labelDiv.className = 'config-field-label';
            
            const labelSpan = document.createElement('span');
            labelSpan.textContent = field.path.split('.').pop();
            labelDiv.appendChild(labelSpan);
            
            if (field.type) {
                const typeTag = document.createElement('span');
                typeTag.className = 'config-field-type-tag';
                typeTag.textContent = `<${field.type}>`;
                labelDiv.appendChild(typeTag);
            }
            
            if (field.isRequired) {
                const requiredSpan = document.createElement('span');
                requiredSpan.className = 'required-indicator';
                requiredSpan.textContent = '*';
                labelDiv.appendChild(requiredSpan);
            }
            
            fieldDiv.appendChild(labelDiv);
            
            // Description as terminal comment style (# description)
            if (field.schema && field.schema.description) {
                const descDiv = document.createElement('div');
                descDiv.className = 'config-field-description-inline';
                descDiv.textContent = '# ' + field.schema.description;
                fieldDiv.appendChild(descDiv);
            } else if (field.schema && !field.schema.description && field.path) {
                // Debug: log when schema exists but description is missing
                console.debug('Field missing description:', field.path, 'schema:', field.schema);
            }
            
            // Field value - neo-brutalist terminal style
            const hasNestedFields = field.nested && Object.keys(field.nested).length > 0;
            const showValue = isConfigured && !(field.type === 'object' && hasNestedFields) && field.type !== 'array';
            
            // Check if value equals default
            const isDefaultValue = field.schema && field.schema.default !== undefined && 
                                   JSON.stringify(field.value) === JSON.stringify(field.schema.default);
            
            const valueDiv = document.createElement('div');
            valueDiv.className = `config-field-value ${isConfigured ? '' : 'unconfigured'} ${isDefaultValue ? 'is-default' : ''}`;
            
            // Handle arrays with explicit container structure
            if (field.type === 'array') {
                const arrayContainer = document.createElement('div');
                arrayContainer.className = 'config-field-container';
                arrayContainer.style.border = 'none';
                
                if (isConfigured && Array.isArray(field.value) && field.value.length > 0) {
                    field.value.forEach((item, index) => {
                        // Create a bordered container for each array item
                        const itemContainer = document.createElement('div');
                        itemContainer.className = 'config-field-array-item';
                        itemContainer.style.border = '2px solid var(--text)';
                        itemContainer.style.padding = '0.5rem';
                        itemContainer.style.marginBottom = '0.5rem';
                        
                        const itemDiv = document.createElement('div');
                        
                        // Check if this array item has nested fields (object type)
                        const hasNestedFieldsForItem = field.nested && Object.keys(field.nested).length > 0;
                        
                        if (hasNestedFieldsForItem && typeof item === 'object' && item !== null) {
                            // Render nested fields for object array items
                            const nestedGroup = document.createElement('div');
                            
                            Object.keys(field.nested).forEach(nestedKey => {
                                const nestedField = field.nested[nestedKey];
                                const nestedValue = item[nestedKey];
                                const nestedIsConfigured = nestedValue !== undefined && nestedValue !== null;
                                const nestedDataPath = `${dataPath}[${index}].${nestedKey}`;
                                
                                // Create nested field item
                                const nestedFieldDiv = this.renderField({
                                    ...nestedField,
                                    path: nestedDataPath,
                                    value: nestedValue,
                                    isConfigured: nestedIsConfigured
                                }, level + 1);
                                
                                nestedGroup.appendChild(nestedFieldDiv);
                            });
                            
                            itemDiv.appendChild(nestedGroup);
                        } else {
                            // Simple array item - render as before
                            const itemLabel = document.createElement('div');
                            itemLabel.textContent = `item ${index}`;
                            itemLabel.style.fontWeight = '600';
                            itemLabel.style.marginBottom = '0.25rem';
                            itemDiv.appendChild(itemLabel);
                            
                            const itemValue = document.createElement('div');
                            itemValue.style.paddingLeft = '2ch';
                            if (typeof item === 'object') {
                                itemValue.textContent = JSON.stringify(item, null, 2);
                            } else {
                                itemValue.textContent = String(item);
                            }
                            itemDiv.appendChild(itemValue);
                        }
                        
                        itemContainer.appendChild(itemDiv);
                        
                        const removeBtn = document.createElement('button');
                        removeBtn.type = 'button';
                        removeBtn.className = 'config-field-remove-btn';
                        removeBtn.textContent = '[ - remove ]';
                        removeBtn.setAttribute('data-field-path', `${dataPath}[${index}]`);
                        itemContainer.appendChild(removeBtn);
                        
                        arrayContainer.appendChild(itemContainer);
                        
                        // Add rule line between items
                        if (index < field.value.length - 1) {
                            const rule = document.createElement('hr');
                            rule.className = 'config-field-rule';
                            arrayContainer.appendChild(rule);
                        }
                    });
                } else {
                    const emptyDiv = document.createElement('div');
                    emptyDiv.className = 'config-field-container-empty';
                    emptyDiv.textContent = '[EMPTY]';
                    arrayContainer.appendChild(emptyDiv);
                }
                
                const addItemBtn = document.createElement('button');
                addItemBtn.type = 'button';
                addItemBtn.className = 'config-field-add-btn';
                addItemBtn.textContent = '[ + add item ]';
                addItemBtn.setAttribute('data-field-path', dataPath);
                arrayContainer.appendChild(addItemBtn);
                
                valueDiv.appendChild(arrayContainer);
            }
            // Handle boolean with text-based selector
            else if (field.type === 'boolean') {
                const booleanDiv = document.createElement('div');
                booleanDiv.className = 'config-field-boolean';
                
                const trueOption = document.createElement('span');
                trueOption.className = 'config-field-boolean-option';
                trueOption.textContent = 'true';
                if (isConfigured && field.value === true) {
                    trueOption.classList.add('selected');
                }
                trueOption.setAttribute('data-field-path', dataPath);
                trueOption.setAttribute('data-value', 'true');
                booleanDiv.appendChild(trueOption);
                
                const separator = document.createTextNode(' | ');
                booleanDiv.appendChild(separator);
                
                const falseOption = document.createElement('span');
                falseOption.className = 'config-field-boolean-option';
                falseOption.textContent = 'false';
                if (isConfigured && field.value === false) {
                    falseOption.classList.add('selected');
                } else if (!isConfigured) {
                    falseOption.classList.add('selected'); // Default to false if not configured
                }
                falseOption.setAttribute('data-field-path', dataPath);
                falseOption.setAttribute('data-value', 'false');
                booleanDiv.appendChild(falseOption);
                
                if (field.schema && field.schema.default !== undefined) {
                    const defaultHint = document.createElement('span');
                    defaultHint.className = 'config-field-default-hint-inline';
                    defaultHint.textContent = `(default: ${field.schema.default})`;
                    booleanDiv.appendChild(defaultHint);
                }
                
                valueDiv.appendChild(booleanDiv);
            }
            // Handle objects with explicit brackets
            else if (field.type === 'object' && hasNestedFields) {
                const objectContainer = document.createElement('div');
                objectContainer.className = 'config-field-container';
                objectContainer.style.border = 'none';
                objectContainer.style.backgroundColor = 'transparent';
                objectContainer.style.color = 'var(--text)';
                objectContainer.style.marginBottom = '0.5rem';
                objectContainer.style.padding = '0.5rem';
                objectContainer.style.fontWeight = '600';
                
                const editBtn = document.createElement('button');
                editBtn.type = 'button';
                editBtn.className = 'config-field-edit-btn';
                editBtn.textContent = '[ edit ]';
                editBtn.setAttribute('data-field-path', dataPath);
                objectContainer.appendChild(editBtn);
                
                valueDiv.appendChild(objectContainer);
            }
            // Handle simple values
            else if (showValue) {
                const valueWrapper = document.createElement('div');
                valueWrapper.style.display = 'flex';
                valueWrapper.style.alignItems = 'center';
                valueWrapper.style.gap = '0.5rem';
                valueWrapper.style.flexWrap = 'wrap';
                
                const formatted = this.formatFieldValue(field.value, field.type);
                const valueSpan = document.createElement('span');
                valueSpan.className = 'config-field-value-content';
                if (typeof formatted === 'string') {
                    valueSpan.textContent = `[ ${formatted} ]`;
                } else {
                    valueSpan.textContent = '[ ';
                    valueSpan.appendChild(formatted);
                    const closing = document.createTextNode(' ]');
                    valueSpan.appendChild(closing);
                }
                if (isDefaultValue) {
                    valueSpan.classList.add('config-field-default-value');
                }
                valueWrapper.appendChild(valueSpan);
                
                // Add default hint inline if value equals default
                if (isDefaultValue && field.schema && field.schema.default !== undefined) {
                    const defaultHint = document.createElement('span');
                    defaultHint.className = 'config-field-default-hint-inline';
                    defaultHint.textContent = '(default)';
                    defaultHint.style.color = 'var(--bg)';
                    defaultHint.style.opacity = '0.7';
                    valueWrapper.appendChild(defaultHint);
                }
                
                // Add Edit button for configured values
                const editButton = document.createElement('button');
                editButton.type = 'button';
                editButton.className = 'config-field-edit-btn';
                editButton.setAttribute('data-field-path', dataPath);
                editButton.textContent = '[ edit ]';
                valueWrapper.appendChild(editButton);
                
                valueDiv.appendChild(valueWrapper);
            } else {
                // For unconfigured fields
                const unconfiguredWrapper = document.createElement('div');
                unconfiguredWrapper.style.display = 'flex';
                unconfiguredWrapper.style.alignItems = 'center';
                unconfiguredWrapper.style.gap = '0.5rem';
                unconfiguredWrapper.style.flexWrap = 'wrap';
                
                const addButton = document.createElement('button');
                addButton.type = 'button';
                addButton.className = 'config-field-add-btn';
                addButton.setAttribute('data-field-path', dataPath);
                addButton.textContent = '[ + add ]';
                unconfiguredWrapper.appendChild(addButton);
                
                // Show default hint inline for unconfigured fields
                if (field.schema && field.schema.default !== undefined) {
                    const defaultHint = document.createElement('span');
                    defaultHint.className = 'config-field-default-hint-inline';
                    defaultHint.textContent = `(default: ${JSON.stringify(field.schema.default)})`;
                    unconfiguredWrapper.appendChild(defaultHint);
                }
                
                valueDiv.appendChild(unconfiguredWrapper);
            }
            
            fieldDiv.appendChild(valueDiv);
            
            // Enum values hint (terminal style)
            if (field.schema && field.schema.enum && Array.isArray(field.schema.enum)) {
                const enumDiv = document.createElement('div');
                enumDiv.className = 'config-field-description-inline';
                enumDiv.textContent = '# Allowed values: ' + field.schema.enum.map(v => String(v)).join(' | ');
                fieldDiv.appendChild(enumDiv);
            }
            
            // Render nested fields with horizontal rules
            if (field.nested && Object.keys(field.nested).length > 0) {
                const nestedGroup = document.createElement('div');
                nestedGroup.className = 'config-nested-group';
                const nestedKeys = Object.keys(field.nested);
                nestedKeys.forEach((key, index) => {
                    const nestedField = this.renderField(field.nested[key], level + 1);
                    nestedGroup.appendChild(nestedField);
                    
                    // Add horizontal rule between nested fields (except last)
                    if (index < nestedKeys.length - 1) {
                        const rule = document.createElement('hr');
                        rule.className = 'config-field-rule';
                        nestedGroup.appendChild(rule);
                    }
                });
                fieldDiv.appendChild(nestedGroup);
            }
            
            return fieldDiv;
        },
        
        formatFieldValue: function(value, type) {
            if (value === null || value === undefined) {
                return 'null';
            }
            
            if (type === 'boolean') {
                return value ? 'true' : 'false';
            }
            
            if (type === 'array' && Array.isArray(value)) {
                return value.map(v => String(v)).join(', ');
            }
            
            if (type === 'object' && typeof value === 'object') {
                return JSON.stringify(value, null, 2);
            }
            
            return String(value);
        },
        
        setupClickHandlers: function() {
            document.querySelectorAll('.config-field-item.unconfigured').forEach(item => {
                const clickHandler = function(e) {
                    if (e.target.classList.contains('config-field-add-btn') || 
                        e.target.classList.contains('config-field-boolean-option')) {
                        e.stopPropagation();
                        return;
                    }
                    const fieldPath = this.getAttribute('data-field-path');
                    DeploymentParams.FieldRenderer.addFieldConfiguration(fieldPath);
                };
                State.addEventListener(item, 'click', clickHandler);
            });
            
            document.querySelectorAll('.config-field-add-btn').forEach(btn => {
                const clickHandler = function(e) {
                    e.preventDefault();
                    e.stopPropagation();
                    const fieldPath = this.getAttribute('data-field-path');
                    DeploymentParams.FieldRenderer.addFieldConfiguration(fieldPath);
                };
                State.addEventListener(btn, 'click', clickHandler);
            });
            
            // Edit button handlers
            document.querySelectorAll('.config-field-edit-btn').forEach(btn => {
                const clickHandler = function(e) {
                    e.preventDefault();
                    e.stopPropagation();
                    const fieldPath = this.getAttribute('data-field-path');
                    DeploymentParams.FieldRenderer.editFieldConfiguration(fieldPath);
                };
                State.addEventListener(btn, 'click', clickHandler);
            });
            
            // Remove button handlers (for array items)
            document.querySelectorAll('.config-field-remove-btn').forEach(btn => {
                const clickHandler = function(e) {
                    e.preventDefault();
                    e.stopPropagation();
                    const fieldPath = this.getAttribute('data-field-path');
                    // TODO: Implement remove array item functionality
                    console.log('Remove item:', fieldPath);
                };
                State.addEventListener(btn, 'click', clickHandler);
            });
            
            // Boolean option handlers
            document.querySelectorAll('.config-field-boolean-option').forEach(option => {
                const clickHandler = function(e) {
                    e.preventDefault();
                    e.stopPropagation();
                    const fieldPath = this.getAttribute('data-field-path');
                    const value = this.getAttribute('data-value') === 'true';
                    
                    // Update visual selection
                    const booleanDiv = this.parentElement;
                    booleanDiv.querySelectorAll('.config-field-boolean-option').forEach(opt => {
                        opt.classList.remove('selected');
                    });
                    this.classList.add('selected');
                    
                    // Navigate to YAML editor to set the value
                    DeploymentParams.FieldRenderer.editFieldConfiguration(fieldPath);
                };
                State.addEventListener(option, 'click', clickHandler);
            });
        },
        
        addFieldConfiguration: function(fieldPath) {
            ViewManager.showYamlEditor();
            
            setTimeout(() => {
                const yamlEditor = State.getYamlEditor();
                if (!yamlEditor) return;
                
                try {
                    const currentYaml = yamlEditor.getValue();
                    let data = {};
                    
                    if (currentYaml && currentYaml.trim()) {
                        try {
                            data = YamlUtils.yamlToJson(currentYaml);
                        } catch (e) {
                            data = {};
                        }
                    }
                    
                    const pathParts = fieldPath.split('.');
                    const fieldSchema = SchemaManager.findFieldSchema(fieldPath);
                    
                    // Create nested path
                    let current = data;
                    for (let i = 0; i < pathParts.length - 1; i++) {
                        const part = pathParts[i];
                        if (!current[part]) {
                            current[part] = {};
                        }
                        current = current[part];
                    }
                    
                    const fieldName = pathParts[pathParts.length - 1];
                    
                    // Determine placeholder value
                    let placeholderValue = '';
                    if (fieldSchema) {
                        if (fieldSchema.default !== undefined) {
                            placeholderValue = fieldSchema.default;
                        } else if (fieldSchema.type === 'string') {
                            placeholderValue = '';
                        } else if (fieldSchema.type === 'integer' || fieldSchema.type === 'number') {
                            placeholderValue = 0;
                        } else if (fieldSchema.type === 'boolean') {
                            placeholderValue = false;
                        } else if (fieldSchema.type === 'object') {
                            placeholderValue = {};
                        } else if (fieldSchema.type === 'array') {
                            placeholderValue = [];
                        }
                    } else {
                        placeholderValue = '';
                    }
                    
                    current[fieldName] = placeholderValue;
                    const newYaml = YamlUtils.jsonToYaml(data);
                    yamlEditor.setValue(newYaml, -1);
                    
                    setTimeout(() => {
                        YamlEditor.scrollToField(fieldPath);
                        yamlEditor.focus();
                    }, 50);
                    
                } catch (error) {
                    alert('Error adding field: ' + error.message);
                }
            }, 100);
        },
        
        editFieldConfiguration: function(fieldPath) {
            ViewManager.showYamlEditor();
            
            setTimeout(() => {
                YamlEditor.scrollToField(fieldPath);
                const yamlEditor = State.getYamlEditor();
                if (yamlEditor) {
                    yamlEditor.focus();
                }
            }, 100);
        }
    };
})();

