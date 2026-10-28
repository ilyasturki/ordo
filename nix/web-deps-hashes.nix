# node_modules holds each platform's native bindings (rolldown, tailwind oxide, lightningcss), so its
# hash differs per system. A system missing here builds ordo with the placeholder page instead of
# the app; `just web-deps-hash` on it adds its hash.
{
  x86_64-linux = "sha256-JdJWqGJpNYqvmkB3jLsBR02UuXsjPzJAGvYtAYid6ZQ=";
}
