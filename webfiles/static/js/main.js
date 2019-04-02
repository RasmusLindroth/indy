class App {
    constructor() {
        this.fetching = false;
        this.fetchQueue = false;
        this.fetchQueueAfter = true;
        this.fetchFail = 0;
        this.loadedAll = false;

        this.isOnline = window.navigator.onLine;

        this.pages = [];
        this.pageObserver = null;
        this.scrollUp = false;


        this.elements = {
            'offline': document.getElementById('offline'),
            'news': document.getElementById('news'),
            'allArticles': document.getElementById('no-more'),
            'loading': document.getElementById('is-loading'),
            'loadingHolder': document.getElementById('loading-holder'),
            'manualFetch': document.getElementById('load-more'),
            'manualFetchBtn': document.getElementById('load-more-btn'),
            'manualFetchPrev': document.getElementById('load-prev-holder'),
            'manualFetchPrevBtn': document.getElementById('load-prev-btn'),
            'pageList': document.getElementById('news-pages'),
        };

        if (this.elements['news'] !== null) {
            this.elements['pageList'].style.display = 'none';

            this.pageStart = parseInt(this.elements['news'].dataset.pageNow);
            this.pageNow = this.pageStart;
            this.currentFakePage = this.pageNow;
            this.pages.push(this.pageNow);
            this.pageTotal = parseInt(this.elements['news'].dataset.pageTotal);
            this.latestArticle = parseInt(this.elements['news'].dataset.latestArticle);

            if (this.pageNow !== 1) {
                this.elements['manualFetchPrev'].style.display = 'block';
            }


            window.addEventListener('online', () => { this.online() });
            window.addEventListener('offline', () => { this.offline() });
            this.elements['manualFetchBtn'].addEventListener('click', () => {
                this.elements['manualFetch'].style.display = 'none';
                this.getNews(true);
            });

            this.elements['manualFetchPrevBtn'].addEventListener('click', () => {
                this.getNews(false);
            });

            if (this.isOnline === false) {
                this.offline();
            }


            this.intersectionObserve();
            this.initPageObserver();
            this.addPageObserver(document.querySelector('.page-item'));
        }
    }

    initPageObserver() {
        var options = {
            root: null,
            rootMargin: '0px 0px -80% 0px',
            threshold: 1,
        };

        this.pageObserver = new IntersectionObserver((entries, observer) => {
            entries.forEach(entry => {
                let target = entry.target;
                let page = parseInt(target.dataset.pageNum);


                if (entry.isIntersecting === true) {
                    history.pushState(null, '', '/news/page/' + page);
                    this.currentFakePage = page;
                } else {

                    let viewportBottom = window.scrollY + window.innerHeight;
                    viewportBottom = viewportBottom - (window.innerHeight * 0.9);
                    let pageTop = target.offsetTop;

                    if (viewportBottom < pageTop && this.pages.indexOf((this.currentFakePage - 1)) !== -1) {
                        this.currentFakePage = this.currentFakePage - 1;
                        history.pushState(null, '', '/news/page/' + this.currentFakePage);
                    }
                }
            });
        }, options);
    }

    addPageObserver(elItem) {
        this.pageObserver.observe(elItem);
    }

    getNews(after) {
        if ((this.loadedAll === true && after) || this.fetching === true) {
            return
        }

        let nextPage = this.pageNow + 1;
        if (after === false) {
            nextPage = nextPage - 2;
        }

        if (nextPage > this.pageTotal) {
            this.reachedEnd();
            return
        }

        if (nextPage < 1) {
            this.loadedFirst()
            return
        }

        this.fetching = true;
        this.isLoading();

        fetch('/api/news/page/' + nextPage)
            .then((response) => {
                if (!response || response.status !== 200) {
                    return Promise.reject('API error');
                }
                return response.json();
            })
            .then((data) => {
                this.gotNews(data, after);
            })
            .catch(error => {
                if (this.isOnline === false) {
                    this.fetchQueue = true;
                    this.fetchQueueAfter = after;
                }
                this.errorNews(after);
            });
    }

    gotNews(data, after) {
        this.fetching = false;
        this.stoppedLoading();
        this.fetchQueue = false;
        this.fetchFail = 0;

        this.pageNow = data['page_now'];
        this.pages.push(this.pageNow);
        this.pageTotal = data['page_total'];

        if (this.pageNow === this.pageTotal) {
            this.reachedEnd();
        }

        if (after === false && this.pageNow === 1) {
            this.loadedFirst();
        }

            this.addPage(data, after);
    }

    addPage(data, after) {
        let sites = data.sites;

        let fragment = document.createDocumentFragment();

        let heading = elementWithClass('h2', 'page-item');
        heading.dataset.pageNum = data['page_now'];
        heading.innerHTML = 'Sida ' + data['page_now'];

        fragment.appendChild(heading);

        for (let i = 0; i < data.articles.length; i++) {
            fragment.appendChild(
                createArticle(data.articles[i], sites)
            );
        }

        if (after) {
            this.elements['news'].appendChild(fragment);
        } else {
            this.elements['news'].prepend(fragment);
        }

        this.addPageObserver(heading);
    }

    errorNews(after) {
        this.fetching = false;
        this.stoppedLoading();
        this.fetchFail++;

        if (this.fetchFail < 3) {
            this.getNews(after);
        } else {
            this.elements['manualFetch'].style.display = 'block';
        }
    }

    loadedFirst() {
        this.elements['manualFetchPrev'].style.display = 'none';
    }

    reachedEnd() {
        this.loadedAll = true;
        this.elements['allArticles'].style.display = 'block';
    }

    isLoading() {
        this.elements['loading'].style.display = 'block';
    }

    stoppedLoading() {
        this.elements['loading'].style.display = 'none';
    }

    online() {
        this.isOnline = true;
        this.elements['offline'].style.display = 'none';

        if (this.fetchQueue === true) {
            this.getNews(this.fetchQueueAfter)
        }
    }

    offline() {
        this.isOnline = false;
        this.elements['offline'].style.display = 'block';
    }

    intersectionObserve() {
        const options = {
            root: null,
            rootMargin: '1000px',
            threshold: 0,
        }

        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    this.getNews(true);
                }
            });
        }, options);

        observer.observe(this.elements['loadingHolder']);
    }
}

/* Helpers */

function createArticle(article, sites) {
    let articleLink = elementWithClass('a', 'news-link');
    articleLink.setAttribute('href', article.URL);
    articleLink.setAttribute('title', article.Title);
    articleLink.setAttribute('rel', 'noreferrer');
    articleLink.setAttribute('target', '_blank');

    let articleEl = elementWithClass('article', 'news-article');
    let publicationDiv = elementWithClass('div', 'news-publication');
    let publicationHeading = document.createElement('h5');
    publicationHeading.innerHTML = sanitize(sites[article.Site]);

    let date = new Date(article.Date);
    let articleDate = elementWithClass('p', 'date');
    articleDate.innerHTML = prettyDate(date);

    publicationDiv.appendChild(publicationHeading);
    publicationDiv.appendChild(articleDate);
    articleEl.appendChild(publicationDiv);

    let articleContent = elementWithClass('div', 'news-content');
    let articleHeading = document.createElement('h3');
    articleHeading.innerHTML = sanitize(article.Title);

    let articleTags = elementWithClass('div', 'tags');
    if (article.Matches & 2 != 0) {
        articleTags.appendChild(tagElement('ros', 'Felix'));
    }
    if (article.Matches & 1 != 0) {
        articleTags.appendChild(tagElement('eri', 'Marcus'));
    }

    articleContent.appendChild(articleHeading);
    articleContent.appendChild(articleTags);
    articleEl.appendChild(articleContent);

    articleLink.appendChild(articleEl);

    return articleLink;
}

function tagElement(abbr, name) {
    let el = elementWithClass('div', 'tag ' + abbr);
    el.innerHTML = name;
    return el;
}

function elementWithClass(element, classes) {
    let el = document.createElement(element);
    el.className = classes;

    return el;
}

function sanitize(str) {
    var el = document.createElement('div');
    el.textContent = str;
    return el.innerHTML;
};

function leadingZero(num) {
    let str = num.toString();
    if (str.length == 1) {
        return "0" + str;
    }
    return str;
}

function prettyDate(date) {
    let y = date.getFullYear();
    let m = leadingZero(date.getMonth());
    let d = leadingZero(date.getDay());
    let hh = leadingZero(date.getHours());
    let mm = leadingZero(date.getMinutes())
    return y + '-' + m + '-' + d + ' ' + hh + ':' + mm;
}

window.addEventListener('DOMContentLoaded', (event) => {
    if ('IntersectionObserver' in window) {
        let app = new App();
    } else {
        var script = document.createElement('script');
        script.onload = function () {
            let app = new App();
        };
        script.src = 'https://polyfill.io/v3/polyfill.js?features=es5,es6,es7,IntersectionObserver&flags=gated';
        document.body.appendChild(script);
    }
});
