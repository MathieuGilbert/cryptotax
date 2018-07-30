if (typeof app === "undefined" || app === null) app = {};
app.files = [];
app.trades = [];
app.newTrade = {
    id: "",
    date: "2017-06-20",
    action: "BUY",
    amount: "10000",
    currency: "OMG",
    baseAmount: "10",
    baseCurrency: "ETH",
    feeAmount: "4",
    feeCurrency: "OMG",
    error: ""
};

$(document).ready(function() {
    $(".navbar-burger").click(function() {
        $(".navbar-burger").toggleClass("is-active");
        $(".navbar-menu").toggleClass("is-active");
    });
});
