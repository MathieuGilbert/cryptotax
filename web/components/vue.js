Vue.component('main-content', {
    data() {
        return { message: "YO" }
    },
});

new Vue({
    delimiters: ['${', '}'],
    el: '#content'
});
