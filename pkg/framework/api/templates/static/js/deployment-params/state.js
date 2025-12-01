// State management for deployment parameters

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    
    const state = {
        instanceData: {},
        deployedData: {},
        services: [],
        crdSchema: null,
        yamlEditor: null,
        originalYaml: '',
        serviceSchemaTemplate: null,
        globalSchema: null,
        eventListeners: [],
        filterNonDefaultK8s: false
    };
    
    DeploymentParams.State = {
        get: function() {
            return state;
        },
        
        getInstanceData: function() {
            return state.instanceData;
        },
        
        setInstanceData: function(data) {
            state.instanceData = data;
        },
        
        getDeployedData: function() {
            return state.deployedData;
        },
        
        setDeployedData: function(data) {
            state.deployedData = data;
        },
        
        getServices: function() {
            return state.services;
        },
        
        setServices: function(services) {
            state.services = services;
        },
        
        getCrdSchema: function() {
            return state.crdSchema;
        },
        
        setCrdSchema: function(schema) {
            state.crdSchema = schema;
        },
        
        getYamlEditor: function() {
            return state.yamlEditor;
        },
        
        setYamlEditor: function(editor) {
            state.yamlEditor = editor;
        },
        
        getOriginalYaml: function() {
            return state.originalYaml;
        },
        
        setOriginalYaml: function(yaml) {
            state.originalYaml = yaml;
        },
        
        getServiceSchemaTemplate: function() {
            return state.serviceSchemaTemplate;
        },
        
        setServiceSchemaTemplate: function(template) {
            state.serviceSchemaTemplate = template;
        },
        
        getGlobalSchema: function() {
            return state.globalSchema;
        },
        
        setGlobalSchema: function(schema) {
            state.globalSchema = schema;
        },
        
        getFilterNonDefaultK8s: function() {
            return state.filterNonDefaultK8s;
        },
        
        setFilterNonDefaultK8s: function(value) {
            state.filterNonDefaultK8s = value;
        },
        
        addEventListener: function(element, event, handler) {
            if (element) {
                element.addEventListener(event, handler);
                state.eventListeners.push({ element, event, handler });
            }
        },
        
        cleanupEventListeners: function() {
            state.eventListeners.forEach(({ element, event, handler }) => {
                element.removeEventListener(event, handler);
            });
            state.eventListeners = [];
        }
    };
})();

