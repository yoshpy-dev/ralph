---
paths:
  - "**/*.dart"
  - "pubspec.yaml"
---
# Dart and Flutter rules

- Follow effective Dart style and lint rules (`flutter analyze` or `dart analyze`).
- Keep widget trees shallow; extract reusable widgets early.
- Use `flutter test` or `dart test` before completion when the project supports them.
- Separate business logic from UI; prefer testable state management over widget-embedded logic.
- Keep platform-specific code behind clear abstraction boundaries.
