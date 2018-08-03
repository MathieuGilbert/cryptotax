Vue.component('report-viewer', {
    data() {
        return {
            report: app.report,
            items: app.reportItems
        }
    },
    methods: {
        setLocale: function(e) {
            switch ($(e.currentTarget).val()) {
            case "CAD":
                this.report.locale = "en-CA";
                break;
            case "USD":
                this.report.locale = "en-US";
                break;
            }
        },
        currency: function(val) {
            var formatter = new Intl.NumberFormat(this.report.locale, {
                style: 'currency',
                currency: this.report.currency,
            });

            return formatter.format(val);
        },
        percent: function(val) {
            var formatter = new Intl.NumberFormat(this.report.locale, {
                style: 'percent',
                minimumFractionDigits: 2,
            });

            return formatter.format(val);
        }
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
            setError(data.error); // TODO: make this work with vue data
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
