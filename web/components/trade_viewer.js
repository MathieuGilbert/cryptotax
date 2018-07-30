Vue.component('trade-viewer', {
    data() {
        return {
            files: app.files,
            trades: app.trades,
        }
    },
    computed: {
        uploadedFiles: function() {
            return this.files.filter(f => f.state == "uploaded");
        }
    },
    methods: {
        getFileTrades: function(e) {
            app.trades.splice(0, app.trades.length);
            
            var s = e.currentTarget;
            if (s.selectedIndex > 0) {
                getFileTrades(s.value);
            }
        }
    }
});

new Vue({
    delimiters: ['${', '}'],
    el: '#tv'
});

function getFileTrades(fid) {
    $.ajax({
        url: '/filetrades?id=' + fid,
        type: 'GET',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        for (var i = 0; i < data.trades.length; i++) {
            app.trades.push(data.trades[i]);
        }
    }).fail(function(e) {
        console.log('failure: ' + e);
    });
}
