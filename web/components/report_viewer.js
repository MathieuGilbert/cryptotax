Vue.component('report-viewer', {
    data() {
        return {
            report: app.report,
            items: app.reportItems,
            rates: app.rates
        }
    },
    methods: {
        setLocale: function(e) {
            switch ($(e.currentTarget).val()) {
            case "CAD":
                this.report.locale = "en-CA";
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
            deep: true
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
            return;
        }

        var ts = Date.now();

        for (var i = 0; i < data.items.length; i++) {
            // need to make a copy, "var" doesn't work with the loop+closure
            let item = data.items[i];

            getRate(item.asset, app.report.currency, ts).then((rate) => {
                var amount = Number(item.amount)
                var value = amount * rate;
                var gain = value / amount - 1;

                app.reportItems.push({
                    asset: item.asset,
                    amount: amount,
                    acb: item.acb,
                    value: value,
                    gain: gain
                });
            });
        }
    }).fail(function(xhr, status, error) {
        console.log("Error loading report");
    });
}

function setError(text) {
    $('.help.is-danger').text(text);
}
