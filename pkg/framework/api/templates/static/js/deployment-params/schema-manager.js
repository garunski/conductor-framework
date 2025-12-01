// Schema management and merging

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    const State = DeploymentParams.State;
    
    DeploymentParams.SchemaManager = {
        extractTemplates: function() {
            const crdSchema = State.getCrdSchema();
            if (!crdSchema || !crdSchema.properties) {
                return;
            }
            
            if (crdSchema.properties.global) {
                State.setGlobalSchema(crdSchema.properties.global);
            }
            
            if (crdSchema.properties.services) {
                const servicesSchema = crdSchema.properties.services;
                
                if (servicesSchema.additionalProperties && typeof servicesSchema.additionalProperties === 'object') {
                    State.setServiceSchemaTemplate(servicesSchema.additionalProperties);
                } else if (servicesSchema.items && servicesSchema.items.properties) {
                    State.setServiceSchemaTemplate(servicesSchema.items);
                } else if (servicesSchema.additionalProperties === true) {
                    State.setServiceSchemaTemplate({ additionalProperties: true });
                }
            }
        },
        
        inferType: function(value) {
            if (value === null) return 'null';
            if (Array.isArray(value)) return 'array';
            if (typeof value === 'object') return 'object';
            if (typeof value === 'boolean') return 'boolean';
            if (typeof value === 'number') {
                return Number.isInteger(value) ? 'integer' : 'number';
            }
            return 'string';
        },
        
        mergeSchemaWithData: function(schema, data, path = '') {
            const merged = {};
            const isServiceField = path.startsWith('services.');
            let allowsAdditional = false;
            
            if (schema) {
                if (schema.additionalProperties === true) {
                    allowsAdditional = true;
                } else if (isServiceField && schema.additionalProperties !== false) {
                    allowsAdditional = true;
                }
            } else if (isServiceField) {
                allowsAdditional = true;
            }
            
            const properties = (schema && schema.properties) || {};
            const required = (schema && schema.required) || [];
            
            // Process schema-defined properties
            Object.keys(properties).forEach(key => {
                const prop = properties[key];
                const fieldPath = path ? `${path}.${key}` : key;
                const dataValue = data && data[key];
                const isConfigured = dataValue !== undefined && dataValue !== null && dataValue !== '';
                
                // Debug: log if description exists in schema
                if (prop.description && path === 'global') {
                    console.debug(`Field ${fieldPath} has description:`, prop.description);
                }
                
                merged[key] = {
                    schema: prop,
                    value: dataValue,
                    isConfigured: isConfigured,
                    isRequired: required.includes(key),
                    path: fieldPath,
                    type: prop.type || this.inferType(dataValue)
                };
                
                // Handle nested objects
                if (prop.type === 'object' && prop.properties) {
                    const nestedData = (data && data[key]) || {};
                    merged[key].nested = this.mergeSchemaWithData(prop, nestedData, fieldPath);
                } else if (prop.type === 'array' && prop.items && prop.items.type === 'object' && prop.items.properties) {
                    const arrayData = (data && data[key]) || [];
                    if (arrayData.length > 0) {
                        merged[key].nested = this.mergeSchemaWithData(prop.items, arrayData[0], `${fieldPath}[]`);
                    } else {
                        merged[key].nested = this.mergeSchemaWithData(prop.items, {}, `${fieldPath}[]`);
                    }
                } else if (prop.type === 'object' && isConfigured && typeof dataValue === 'object' && dataValue !== null && !Array.isArray(dataValue)) {
                    const nestedSchema = prop.properties ? prop : { additionalProperties: true };
                    merged[key].nested = this.mergeSchemaWithData(nestedSchema, dataValue, fieldPath);
                } else if (!prop.type && isConfigured && typeof dataValue === 'object' && dataValue !== null && !Array.isArray(dataValue)) {
                    merged[key].nested = this.mergeSchemaWithData({ additionalProperties: true }, dataValue, fieldPath);
                }
            });
            
            // Include fields from data that aren't in schema if allowed
            if (allowsAdditional && data && typeof data === 'object' && !Array.isArray(data)) {
                Object.keys(data).forEach(key => {
                    if (merged[key]) return;
                    
                    const dataValue = data[key];
                    const fieldPath = path ? `${path}.${key}` : key;
                    const isConfigured = dataValue !== undefined && dataValue !== null && dataValue !== '';
                    const inferredType = this.inferType(dataValue);
                    
                    merged[key] = {
                        schema: { type: inferredType },
                        value: dataValue,
                        isConfigured: isConfigured,
                        isRequired: false,
                        path: fieldPath,
                        type: inferredType
                    };
                    
                    if (inferredType === 'object' && !Array.isArray(dataValue) && dataValue !== null) {
                        const nestedResult = this.mergeSchemaWithData({ additionalProperties: true }, dataValue, fieldPath);
                        merged[key].nested = nestedResult;
                    } else if (inferredType === 'array' && Array.isArray(dataValue) && dataValue.length > 0 && typeof dataValue[0] === 'object') {
                        merged[key].nested = this.mergeSchemaWithData({ additionalProperties: true }, dataValue[0], `${fieldPath}[]`);
                    }
                });
            }
            
            return merged;
        },
        
        findFieldSchema: function(fieldPath) {
            const crdSchema = State.getCrdSchema();
            const serviceSchemaTemplate = State.getServiceSchemaTemplate();
            const pathParts = fieldPath.split('.');
            let currentSchema = crdSchema;
            let fieldSchema = null;
            
            for (let i = 0; i < pathParts.length && currentSchema; i++) {
                const part = pathParts[i];
                
                if (part === 'global' && currentSchema.properties && currentSchema.properties.global) {
                    currentSchema = currentSchema.properties.global;
                } else if (part === 'services' && currentSchema.properties && currentSchema.properties.services) {
                    currentSchema = currentSchema.properties.services;
                } else if (i === 1 && pathParts[0] === 'services' && part !== 'services') {
                    if (serviceSchemaTemplate) {
                        currentSchema = serviceSchemaTemplate;
                    } else if (currentSchema.additionalProperties && typeof currentSchema.additionalProperties === 'object') {
                        currentSchema = currentSchema.additionalProperties;
                    } else {
                        currentSchema = null;
                    }
                } else if (currentSchema.additionalProperties && typeof currentSchema.additionalProperties === 'object') {
                    if (currentSchema.properties && currentSchema.properties[part]) {
                        currentSchema = currentSchema.properties[part];
                    } else if (currentSchema.additionalProperties.properties && currentSchema.additionalProperties.properties[part]) {
                        currentSchema = currentSchema.additionalProperties.properties[part];
                    } else {
                        currentSchema = { type: 'string' };
                    }
                } else if (currentSchema.properties && currentSchema.properties[part]) {
                    currentSchema = currentSchema.properties[part];
                } else {
                    currentSchema = null;
                }
                
                if (i === pathParts.length - 1) {
                    fieldSchema = currentSchema;
                }
            }
            
            return fieldSchema;
        }
    };
})();

