// YAML/JSON conversion utilities

(function() {
    'use strict';
    
    window.DeploymentParams = window.DeploymentParams || {};
    
    DeploymentParams.YamlUtils = {
        jsonToYaml: function(obj) {
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
                return '';
            }
        },
        
        yamlToJson: function(yamlStr) {
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
    };
})();

