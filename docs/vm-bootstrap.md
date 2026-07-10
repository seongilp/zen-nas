# 빌드 VM 첫 부팅 절차 (최초 1회 필수)

> 새 VM(`scripts/vm-setup.sh` 직후)은 SSH가 꺼져 있어 **이 절차를 한 번 수행해야
> 이후의 모든 빌드 스크립트가 동작한다.** 이미 SSH가 되는 VM이라면 건너뛴다.

1. `scripts/vm-run.sh` 실행. 부팅 로그가 흐르고 `login:` 프롬프트가 나온다.
   첫 부팅은 파일시스템 자동 확장(growfs) 때문에 조금 더 걸린다.
2. `root` 입력 (비밀번호 없음).
3. 아래를 순서대로 실행:

   ```sh
   sysrc sshd_enable=YES
   mkdir -p /root/.ssh && chmod 700 /root/.ssh
   ```

4. 맥의 **다른 터미널**에서 공개키를 복사해 둔다:

   ```sh
   cat ~/.ssh/id_ed25519.pub   # 없으면: ssh-keygen -t ed25519
   ```

5. VM 콘솔에서 그 키를 등록:

   ```sh
   echo '<복사한 공개키 한 줄>' > /root/.ssh/authorized_keys
   chmod 600 /root/.ssh/authorized_keys
   printf 'PermitRootLogin prohibit-password\n' >> /etc/ssh/sshd_config
   service sshd start
   ```

6. 맥에서 접속 확인:

   ```sh
   ssh -p 10022 root@127.0.0.1 uname -mr
   ```

   → `15.1-RELEASE arm64` 형태가 나오면 성공.

- 콘솔 로그인에 비밀번호는 필요 없다(공식 VM 이미지 기본). SSH는 키 인증만 허용된다.
- VM 종료는 콘솔 또는 ssh에서 `poweroff`.
- `scripts/vm-run.sh -d` 로 데몬 모드 기동도 가능하다. 이때 콘솔은 `nc -U vm/console.sock` 으로 접속한다.
