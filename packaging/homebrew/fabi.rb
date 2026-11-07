# frozen_string_literal: true

# Homebrew formula for fabi — drop this file into a `homebrew-fabi`
# tap repo at Formula/fabi.rb, then users run:
#   brew install shizukutanaka/fabi/fabi
class Fabi < Formula
  desc "Universal AI Control Plane — the honest eye on your AI estate"
  homepage "https://github.com/shizukutanaka/fabi"
  version "0.0.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/shizukutanaka/fabi/releases/download/v0.0.5/fabi-darwin-amd64"
      sha256 "65555cb90a35ac52f8d64ecb7cf921337bb062cb773c1732a26ea53a15303661"
    else
      url "https://github.com/shizukutanaka/fabi/releases/download/v0.0.5/fabi-darwin-arm64"
      sha256 "df3ae798980ded59621aa17962871dfca30531cf02262b2800406485024ce337"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/shizukutanaka/fabi/releases/download/v0.0.5/fabi-linux-amd64"
      sha256 "73634f3697eef8549073efda8c254d699200fc1dfc9c984d5e969910fcde2ec5"
    end
  end

  def install
    bin.install Dir["fabi-*"].first => "fabi"
  end

  test do
    assert_match "fabi", shell_output("#{bin}/fabi version")
  end
end
