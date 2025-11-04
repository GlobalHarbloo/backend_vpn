import 'package:flutter/material.dart';
import '../../../vpn/presentation/pages/vpn_page.dart';
import '../../../traffic/presentation/pages/traffic_page.dart';
import 'profile_page.dart';
// payments page is temporarily removed until payment flow is ready

class MainNavPage extends StatefulWidget {
  const MainNavPage({super.key});

  @override
  State<MainNavPage> createState() => _MainNavPageState();
}

class _MainNavPageState extends State<MainNavPage> {
  int _currentIndex = 0;
  final ValueNotifier<int> _tabNotifier = ValueNotifier<int>(0);

  late final List<Widget> _pages;

  @override
  void initState() {
    super.initState();
    _pages = [
      const VpnPage(),
      const TrafficPage(),
      ProfilePage(tabNotifier: _tabNotifier, tabIndex: 2),
    ];
  }

  @override
  void dispose() {
    _tabNotifier.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(index: _currentIndex, children: _pages),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: _currentIndex,
        onTap: (index) {
          setState(() => _currentIndex = index);
          _tabNotifier.value = index;
        },
        items: const [
          BottomNavigationBarItem(icon: Icon(Icons.vpn_lock), label: 'VPN'),
          BottomNavigationBarItem(icon: Icon(Icons.bar_chart), label: 'Трафик'),
          BottomNavigationBarItem(icon: Icon(Icons.person), label: 'Профиль'),
        ],
      ),
    );
  }
}
