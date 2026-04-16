# Dart / Flutter pack

Default verification order:
- dart format --set-exit-if-changed
- dart analyze (or flutter analyze)
- dart test (or flutter test)

Customize this pack if your repo uses:
- Flutter-specific integration tests (integration_test/)
- custom analysis_options.yaml with strict rules
- build_runner or code generation (freezed, json_serializable)
- melos for monorepo management
- very_good_analysis or other lint packages
