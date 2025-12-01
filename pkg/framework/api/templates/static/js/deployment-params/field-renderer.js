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
            header.textContent = title;
            groupDiv.appendChild(header);
            
            const body = document.createElement('div');
            body.className = 'config-field-group-body';
            
            Object.keys(mergedFields).forEach(key => {
                const field = mergedFields[key];
                const fieldElement = this.renderField(field, 0);
                body.appendChild(fieldElement);
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
            
            // Field label
            const labelDiv = document.createElement('div');
            labelDiv.className = 'config-field-label';
            
            const labelSpan = document.createElement('span');
            labelSpan.textContent = field.path.split('.').pop();
            labelDiv.appendChild(labelSpan);
            
            if (field.type) {
                const badge = document.createElement('span');
                badge.className = `badge ${this.getTypeBadgeColor(field.type)}`;
                badge.textContent = field.type;
                labelDiv.appendChild(badge);
            }
            
            if (field.isRequired) {
                const requiredSpan = document.createElement('span');
                requiredSpan.className = 'required-indicator';
                requiredSpan.textContent = '*';
                labelDiv.appendChild(requiredSpan);
            }
            
            fieldDiv.appendChild(labelDiv);
            
            // Field description (show right after label for better visibility)
            if (field.schema && field.schema.description) {
                const descDiv = document.createElement('div');
                descDiv.className = 'config-field-description';
                descDiv.textContent = field.schema.description;
                fieldDiv.appendChild(descDiv);
            } else if (field.schema && !field.schema.description && field.path) {
                // Debug: log when schema exists but description is missing
                console.debug('Field missing description:', field.path, 'schema:', field.schema);
            }
            
            // Field value
            const hasNestedFields = field.nested && Object.keys(field.nested).length > 0;
            const showValue = isConfigured && !(field.type === 'object' && hasNestedFields);
            
            const valueDiv = document.createElement('div');
            valueDiv.className = `config-field-value ${isConfigured ? '' : 'unconfigured'}`;
            
            if (showValue) {
                const formatted = this.formatFieldValue(field.value, field.type);
                if (typeof formatted === 'string') {
                    valueDiv.textContent = formatted;
                } else {
                    valueDiv.appendChild(formatted);
                }
            } else if (isConfigured && field.type === 'object' && hasNestedFields) {
                const summary = document.createElement('span');
                summary.className = 'text-muted';
                summary.textContent = `Object with ${Object.keys(field.nested).length} field(s)`;
                valueDiv.appendChild(summary);
            } else {
                const notConfigured = document.createElement('span');
                notConfigured.className = 'text-muted';
                notConfigured.textContent = 'Not configured';
                valueDiv.appendChild(notConfigured);
                
                const addLink = document.createElement('a');
                addLink.href = '#';
                addLink.className = 'config-field-add-btn';
                addLink.setAttribute('data-field-path', dataPath);
                addLink.textContent = '[+ Add]';
                valueDiv.appendChild(addLink);
            }
            
            fieldDiv.appendChild(valueDiv);
            
            // Default value hint
            if (!isConfigured && field.schema && field.schema.default !== undefined) {
                const defaultDiv = document.createElement('div');
                defaultDiv.className = 'config-field-description text-info';
                const defaultText = document.createTextNode('Default: ');
                const defaultCode = document.createElement('code');
                defaultCode.textContent = JSON.stringify(field.schema.default);
                defaultDiv.appendChild(defaultText);
                defaultDiv.appendChild(defaultCode);
                fieldDiv.appendChild(defaultDiv);
            }
            
            // Enum values hint
            if (field.schema && field.schema.enum && Array.isArray(field.schema.enum)) {
                const enumDiv = document.createElement('div');
                enumDiv.className = 'config-field-description';
                const enumText = document.createTextNode('Allowed values: ');
                enumDiv.appendChild(enumText);
                field.schema.enum.forEach((v, i) => {
                    if (i > 0) {
                        enumDiv.appendChild(document.createTextNode(', '));
                    }
                    const code = document.createElement('code');
                    code.textContent = String(v);
                    enumDiv.appendChild(code);
                });
                fieldDiv.appendChild(enumDiv);
            }
            
            // Render nested fields
            if (field.nested && Object.keys(field.nested).length > 0) {
                const nestedGroup = document.createElement('div');
                nestedGroup.className = 'config-nested-group';
                Object.keys(field.nested).forEach(key => {
                    const nestedField = this.renderField(field.nested[key], level + 1);
                    nestedGroup.appendChild(nestedField);
                });
                fieldDiv.appendChild(nestedGroup);
            }
            
            return fieldDiv;
        },
        
        formatFieldValue: function(value, type) {
            if (value === null || value === undefined) {
                const span = document.createElement('span');
                span.className = 'text-muted';
                span.textContent = '—';
                return span;
            }
            
            if (type === 'boolean') {
                return value ? 'true' : 'false';
            }
            
            if (type === 'array' && Array.isArray(value)) {
                return value.map(v => String(v)).join(', ');
            }
            
            if (type === 'object' && typeof value === 'object') {
                const code = document.createElement('code');
                code.textContent = JSON.stringify(value, null, 2);
                return code;
            }
            
            return String(value);
        },
        
        getTypeBadgeColor: function(type) {
            const colors = {
                'string': 'bg-primary',
                'integer': 'bg-info',
                'number': 'bg-info',
                'boolean': 'bg-success',
                'array': 'bg-warning',
                'object': 'bg-secondary'
            };
            return colors[type] || 'bg-secondary';
        },
        
        setupClickHandlers: function() {
            document.querySelectorAll('.config-field-item.unconfigured').forEach(item => {
                const clickHandler = function(e) {
                    if (e.target.classList.contains('config-field-add-btn')) {
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
        }
    };
})();

