1. поды

```bash
kubectl --context=sre -n wishlist get pods
kubectl --context=sre -n monitoring get pods
```

2. графики в норме  

3. задержка и ошибки

```bash
kubectl --context=sre -n wishlist set env deployment/wishlist-service \
  DEMO_DELAY=1s DEMO_ERROR_RATE=1

kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

4. через 90 секунд: графики 1, 2 и 4 просели

```bash
kubectl --context=sre -n wishlist logs deployment/wishlist-prober --tail=5
```

5. убрать задержку и ошибки

```bash
kubectl --context=sre -n wishlist set env deployment/wishlist-service \
  DEMO_DELAY=0s DEMO_ERROR_RATE=0

kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

6. остановить бд

```bash
kubectl --context=sre -n wishlist scale statefulset/postgres --replicas=0
kubectl --context=sre -n wishlist wait --for=delete pod/postgres-0 --timeout=60s
```

7. через 90 секунд: третий график просел

```bash
kubectl --context=sre -n wishlist logs deployment/wishlist-prober --tail=5
```

8. вернуть бд

```bash
kubectl --context=sre -n wishlist scale statefulset/postgres --replicas=1
kubectl --context=sre -n wishlist rollout status statefulset/postgres --timeout=180s
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

9. через 90 секунд: проверить восстановление

```bash
kubectl --context=sre -n wishlist logs deployment/wishlist-prober --tail=5

curl -sS --get http://localhost:8428/api/v1/query \
  --data-urlencode 'query=last_over_time(wishlist_probe_success{namespace="wishlist",app="wishlist-prober"}[30s])'
```

10. показать историю, возврата к 100% не ждать
