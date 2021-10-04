"use strict";

module.exports = function (grunt) {
  // Load all grunt tasks matching the ['grunt-*', '@*/grunt-*'] patterns
  require("load-grunt-tasks")(grunt);

  grunt.initConfig({
    bowercopy: {
      options: {
        clean: false, // Bower components folder will be removed afterwards
      },
      libs: {
        options: {
          destPrefix: "static/js/ext",
        },
        files: {
          "jquery.js": "jquery/dist/jquery.js",
          "bootstrap.js": "bootstrap/dist/js/bootstrap.js",
          "twemoji.js": "twemoji/v/13.1.0/twemoji.js",
        },
      },
      css: {
        options: {
          destPrefix: "static/css/ext",
        },
        files: {
          "bootstrap.css": "bootstrap/dist/css/bootstrap.css",
        },
      },
      emojis: {
        options: {
          destPrefix: "static/images/emojis",
        },
        files: {
          "72x72": "twemoji/v/13.1.0/72x72",
          svg: "twemoji/v/13.1.0/svg",
        },
      },
      fonts: {
        options: {
          destPrefix: "static/fonts",
        },
        files: {
          roboto: "roboto-googlefont/*.ttf",
        },
      },
      swagger: {
        options: {
          destPrefix: "swagger",
        },
        files: {
          "favicon-16x16.png": "swagger-ui/dist/favicon-16x16.png",
          "favicon-32x32.png": "swagger-ui/dist/favicon-32x32.png",
          "index.html": "swagger-ui/dist/index.html",
          "swagger-ui.css": "swagger-ui/dist/swagger-ui.css",
          "swagger-ui-bundle.js": "swagger-ui/dist/swagger-ui-bundle.js",
          "swagger-ui-standalone-preset.js":
            "swagger-ui/dist/swagger-ui-standalone-preset.js",
        },
      },
    },
  });

  grunt.registerTask("default", ["bowercopy"]);
};
