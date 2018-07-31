Vue.component('trade-manager', {
    data() {
        return {
            trades: app.trades,
            newTrade: app.newTrade
        }
    },
    computed: {
        disableAdd: function() {
            return error(this.newTrade) !== "";
        }
    },
    methods: {
        clearRow: function(e) {
            resetTrade(app.newTrade);
        },
        addTrade: function(e) {
            var t = app.newTrade;
            var err = error(t);
            t.error = err;

            if (err === "") {
                saveTrade(t);
            }
        },
        deleteTrade: function(e, trade) {
            deleteTrade(trade.id);
            this.toggleDelete(e);
        },
        toggleDelete: function(e) {
            var row = $(e.currentTarget).closest("tr")
            row.find(".delete-button").toggleClass("hidden");
            row.find(".confirm-button").toggleClass("hidden");
            row.find(".keep-button").toggleClass("hidden");
        },
        shortDate: function(date) {
            return formatDate(date);
        },
        longDate: function(date) {
            return formatDateLong(date);
        },
        isValid: function(e) {
            // basic validation
            var input = $(e.currentTarget);

            if (input.val() === "") {
                input.addClass("is-danger");
            } else {
                input.removeClass("is-danger");
            }
        }
    }
});

new Vue({
    delimiters: ['${', '}'],
    el: '#tm'
});

function resetTrade(t) {
    var n = newTrade();

    Object.keys(n).forEach(function(key, i) {
        t[key] = n[key];
    });
}

function saveTrade(trade) {
    var data = JSON.stringify({
        trade: trade,
        CSRFToken: $('input[name="csrf_token"]').val()
    });

    $.ajax({
        url: '/trade',
        type: 'POST',
        data: data,
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        data.trade.date = formatDate(data.trade.date);
        app.trades.push(data.trade);
        resetTrade(app.newTrade);
    }).fail(function(e) {
        app.newTrade.error = "Couldn't save trade.";
    });
}

function deleteTrade(tid) {
    var tindex = app.trades.findIndex(t => t.id === tid);
    var url = '/trade?id=' + tid + '&csrf_token=' + $('input[name="csrf_token"]').val();

    $.ajax({
        url: url,
        type: 'DELETE',
        cache: false,
        contentType: false,
        processData: false,
        timeout: 5000
    }).done(function(data) {
        app.trades.splice(tindex, 1);
    }).fail(function(e) {
        console.log("Error delting trade id: " + tid);
    });
}

function error(t) {
    if (t.date === undefined || t.date.length !== 10) {
        return "Invalid date.";
    }
    if (t.action === undefined || (t.action !== "BUY" && t.action !== "SELL")) {
        return "Must be BUY or SELL.";
    }
    if (t.amount === undefined || t.amount.length === 0) {
        return "Amount missing.";
    }
    if (t.currency === undefined || t.currency.length === 0) {
        return "Currency missing.";
    }
    if (t.baseAmount === undefined || t.baseAmount.length === 0) {
        return "For amount missing.";
    }
    if (t.baseCurrency === undefined || t.baseCurrency.length === 0) {
        return "For currency missing.";
    }
    if (t.feeAmount === undefined || t.feeAmount.length === 0) {
        return "Fee amount missing.";
    }
    if (t.feeCurrency === undefined || t.feeCurrency.length === 0) {
        return "Fee currency missing.";
    }

    return "";
}
