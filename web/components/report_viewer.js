Vue.component('report-viewer', {
    data() {
        return {
            report: app.report,
            items: app.reportItems
        }
    },
    methods: {

    },
    watch: {
        report: {
            handler: function(report) {
                if (report.type !== "" && report.currency !== "" && report.asOf !== "") {
                    loadReport(report);
                }
            },
            deep: true,
        }
    }
});

new Vue({
    delimiters: ['${', '}'],
    el: '#rv'
});

function loadReport(report) {
    app.reportItems.splice(0, app.reportItems.length);
    setError("");

    var url = '/report?type=' + report.type + '&currency=' + report.currency + '&asof=' + report.asOf;
    url += '&csrf_token=' + $('input[name="csrf_token"]').val();

    $.ajax({
        url: url,
        type: 'GET',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        if (data.error.length) {
            setError(data.error);
        } else {
            for (var i = 0; i < data.items.length; i++) {
                app.reportItems.push(data.items[i]);
            }
        }
    }).fail(function(xhr, status, error) {
        console.log("Error loading report");
    });
}

function setError(text) {
    $('.help.is-danger').text(text);
}
