var CACHE_NAME = 'indycar-v1';

var urlsToCache = [
    '/',
    '/error/offline',
];

self.addEventListener('install', function (event) {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(function (cache) {
                return cache.addAll(urlsToCache);
            })
    );
});

self.addEventListener('fetch', function (event) {
    event.respondWith(

        fetch(event.request)
            .then(async function (response) {
                if (!response || (response.status !== 200 && response.status !== 404) || response.type !== 'basic') {
                    return response;
                }

                var responseToCache = response.clone();

                console.log("Adding: " + event.request)
                const cache = await caches.open(CACHE_NAME);
                cache.put(event.request, responseToCache);
                return response;
            })
            .catch(async function () {
                const cache = await caches.open(CACHE_NAME);
                return cache.match(event.request).then(function(response) {
                    if(!response) {
                        return cache.match('/error/offline');
                    }else {
                        return response;
                    }
                });
            })

    );
});
