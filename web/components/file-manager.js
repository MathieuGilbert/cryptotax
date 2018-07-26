Vue.component('file-manager', {
    data() {
        return {
            files: app.files
        }
    },
    methods: {
        removeFile: function(e) {
            var i = $('.remove-file').index(e.currentTarget);
            app.files.splice(i, 1);
        }
    }
});

new Vue({
    delimiters: ['${', '}'],
    el: '#content'
});
